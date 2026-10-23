// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/rogpeppe/go-internal/lockedfile"
)

// The dedupe store is a content-addressed directory of hard links, modelled
// on pnpm's and uv's stores:
//
//	~/.zvm/store/<first two hex chars>/<sha256>-<octal perm>[-<uid>.<gid>]
//
// Each entry is one more hard link to a file that also lives in one or more
// version directories. The store never holds the only copy of anything a
// version needs: removing a store entry only removes a name, never data that a
// version directory still links to. That is what makes both `zvm dedupe` and
// the garbage collector safe to interrupt at any point.
//
// Permission bits (and, on Unix, the owner) are part of the key because hard
// links share one inode, and therefore one mode and one owner. Merging a 0644
// file with a 0755 file would silently change one of them.
const (
	storeDirName = "store"

	// dedupeTempPrefix names the temporary hard link created next to a file
	// before it is atomically renamed over that file. A leftover one (from an
	// interrupted run) is just an extra link and is removed on the next run.
	dedupeTempPrefix = ".zvm-dedupe-"

	// maxConsecutiveLinkFailures is how many files in a row may fail to hard
	// link before dedupe stops trying in that version directory. It keeps a
	// filesystem without hard links (FAT, some network mounts) or a version
	// directory on another device from being hashed in full for nothing.
	maxConsecutiveLinkFailures = 16
)

// isReservedName reports whether name is one of ZVM's own top-level
// directories, which must never be listed, used, aliased or uninstalled as a
// Zig version.
func isReservedName(name string) bool {
	// Case-insensitive, like checkReservedVersion: on macOS and Windows
	// "STORE" names the same directory as "store".
	switch strings.ToLower(name) {
	case "bin", "self", storeDirName:
		return true
	}
	return false
}

// fileID identifies a file on disk independently of its path (device + inode
// on Unix, volume serial + file index on Windows).
type fileID struct {
	dev, ino uint64
}

// Seams for tests, which need to simulate EXDEV and failures mid-replace.
var (
	linkFile   = os.Link
	renameFile = os.Rename
)

// DedupeReport summarizes one `zvm dedupe` run.
type DedupeReport struct {
	Versions []string

	// BytesBefore and BytesAfter are the bytes held by regular files in all
	// installed version directories plus the store, counting each inode once.
	BytesBefore int64
	BytesAfter  int64

	Scanned       int // regular files examined
	Stored        int // first copy of its content seen: added to the store
	Linked        int // replaced with a hard link to an identical store entry
	AlreadyLinked int // already a hard link to its store entry (re-run)
	Ignored       int // empty files and setuid/setgid/sticky files
	Skipped       int // could not be linked; left exactly as they were

	// Problems holds one line per skipped file or repaired store entry.
	Problems []string
}

// BytesSaved is how much smaller the installs plus store became.
func (r DedupeReport) BytesSaved() int64 {
	return r.BytesBefore - r.BytesAfter
}

// PruneReport summarizes a store garbage collection.
type PruneReport struct {
	Removed    int
	BytesFreed int64
}

func (z *ZVM) storeDir() string {
	return filepath.Join(z.baseDir, storeDirName)
}

// lockStore serializes version-file changes with dedupe and store pruning.
// Keep the lock file in place: removing it would let processes lock different
// inodes. The returned unlock is safe to call before a deferred unlock.
func (z *ZVM) lockStore() (func(), error) {
	unlock, err := lockedfile.MutexAt(filepath.Join(z.baseDir, "store.lock")).Lock()
	if err != nil {
		return nil, err
	}
	return sync.OnceFunc(unlock), nil
}

// Dedupe hard-links identical files across installed versions through the
// store. With no versions given it processes every installed version.
func (z *ZVM) Dedupe(versions []string) (DedupeReport, error) {
	unlock, err := z.lockStore()
	if err != nil {
		return DedupeReport{}, errors.Join(ErrDedupeFailed, err)
	}
	defer unlock()

	installed, err := z.GetInstalledVersions()
	if err != nil {
		return DedupeReport{}, errors.Join(ErrDedupeFailed, err)
	}

	if len(versions) == 0 {
		versions = installed
	} else {
		resolved := make([]string, 0, len(versions))
		for _, v := range versions {
			if isReservedName(v) {
				return DedupeReport{}, fmt.Errorf("%w: %q", ErrReservedName, v)
			}
			if r, err := resolveVersionShorthand(v, installed); err == nil {
				v = r
			}
			if !slices.Contains(installed, v) {
				return DedupeReport{}, fmt.Errorf("%w: %q", ErrVersionNotInstalled, v)
			}
			resolved = append(resolved, v)
		}
		versions = resolved
	}

	report := DedupeReport{Versions: versions}
	if report.BytesBefore, err = z.diskUsage(); err != nil {
		return report, errors.Join(ErrDedupeFailed, err)
	}

	if err := os.MkdirAll(z.storeDir(), 0755); err != nil {
		return report, errors.Join(ErrDedupeFailed, err)
	}

	d := &deduper{store: z.storeDir(), report: &report, verified: map[string]fs.FileInfo{}}
	for _, v := range versions {
		if err := d.dedupeTree(filepath.Join(z.baseDir, v)); err != nil {
			return report, errors.Join(ErrDedupeFailed, err)
		}
	}

	if report.BytesAfter, err = z.diskUsage(); err != nil {
		return report, errors.Join(ErrDedupeFailed, err)
	}
	return report, nil
}

type deduper struct {
	store  string
	report *DedupeReport
	// verified caches store entries whose content has been re-hashed this run.
	verified map[string]fs.FileInfo
}

type dedupeOutcome int

const (
	outcomeIgnored dedupeOutcome = iota
	outcomeStored
	outcomeLinked
	outcomeAlreadyLinked
	outcomeSkipped     // a problem unrelated to hard-link support
	outcomeLinkFailure // os.Link or the replacing rename failed
)

func (d *deduper) problem(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Debug("dedupe", "problem", msg)
	d.report.Problems = append(d.report.Problems, msg)
}

func (d *deduper) dedupeTree(root string) error {
	failStreak := 0

	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			d.problem("%s: %v", path, err)
			d.report.Skipped++
			return nil // WalkDir skips an unreadable directory's contents
		}

		// WalkDir never follows symlinks, so only real directories are entered.
		if entry.IsDir() {
			return nil
		}

		if strings.HasPrefix(entry.Name(), dedupeTempPrefix) {
			// Left by an interrupted run: only ever an extra hard link.
			if err := os.Remove(path); err != nil {
				d.problem("%s: removing leftover temp link: %v", path, err)
			}
			return nil
		}

		// Skip symlinks, devices, sockets, ... — only regular files.
		if !entry.Type().IsRegular() {
			return nil
		}

		d.report.Scanned++
		switch outcome := d.dedupeFile(path); outcome {
		case outcomeIgnored:
			d.report.Ignored++
		case outcomeStored:
			d.report.Stored++
			failStreak = 0
		case outcomeLinked:
			d.report.Linked++
			failStreak = 0
		case outcomeAlreadyLinked:
			d.report.AlreadyLinked++
			failStreak = 0
		case outcomeSkipped:
			d.report.Skipped++
		case outcomeLinkFailure:
			d.report.Skipped++
			failStreak++
			if failStreak >= maxConsecutiveLinkFailures {
				d.problem("%s: %d hard links in a row failed; leaving the rest of this version untouched (filesystem without hard links, or on a different device than %s?)",
					root, failStreak, d.store)
				return fs.SkipAll
			}
		}
		return nil
	})
}

func (d *deduper) dedupeFile(path string) dedupeOutcome {
	info, err := os.Lstat(path)
	if err != nil {
		d.problem("%s: %v", path, err)
		return outcomeSkipped
	}
	mode := info.Mode()
	if !mode.IsRegular() {
		return outcomeIgnored
	}
	// Empty files save nothing; special mode bits are not worth reasoning about.
	if info.Size() == 0 || mode&(fs.ModeSetuid|fs.ModeSetgid|fs.ModeSticky) != 0 {
		return outcomeIgnored
	}

	sum, err := hashFile(path)
	if err != nil {
		d.problem("%s: %v", path, err)
		return outcomeSkipped
	}

	// Refuse to store a file that changed while it was being hashed.
	if after, err := os.Lstat(path); err != nil || !unchangedStoreFile(info, after) {
		d.problem("%s: changed while hashing; left untouched", path)
		return outcomeSkipped
	}

	name := storeEntryName(sum, info)
	entry := filepath.Join(d.store, name[:2], name)

	// Two attempts: a lost race creating the entry, or a stale entry that had
	// to be dropped, sends us round once more.
	for range 2 {
		sinfo, err := os.Lstat(entry)
		if errors.Is(err, fs.ErrNotExist) {
			if err := os.MkdirAll(filepath.Dir(entry), 0755); err != nil {
				d.problem("%s: %v", entry, err)
				return outcomeSkipped
			}
			// The file itself becomes the store copy: one extra link, no copy.
			if err := linkFile(path, entry); err != nil {
				if errors.Is(err, fs.ErrExist) {
					continue
				}
				d.problem("%s: cannot hard link into store: %v", path, err)
				return outcomeLinkFailure
			}
			linked, err := os.Lstat(entry)
			if err != nil || !unchangedStoreFile(info, linked) {
				_ = os.Remove(entry)
				d.problem("%s: changed before linking into store; left untouched", path)
				return outcomeSkipped
			}
			d.verified[entry] = linked
			return outcomeStored
		}
		if err != nil {
			d.problem("%s: %v", entry, err)
			return outcomeSkipped
		}

		if os.SameFile(info, sinfo) {
			return outcomeAlreadyLinked
		}

		if !d.verify(entry, sinfo, sum, mode) {
			// The entry no longer matches its name, so a file linked to it was
			// edited (or chmod-ed) in place — which also changed every version
			// sharing it. Drop the store's name; the data stays reachable from
			// those versions, and this file seeds a fresh entry.
			d.problem("%s: store entry no longer matches its hash or mode — a deduped file was modified in place, changing every version linked to it; consider `zvm install --force` for affected versions", entry)
			if err := os.Remove(entry); err != nil && !errors.Is(err, fs.ErrNotExist) {
				d.problem("%s: %v", entry, err)
				return outcomeSkipped
			}
			delete(d.verified, entry)
			continue
		}

		if err := replaceWithLink(entry, path); err != nil {
			d.problem("%s: cannot replace with hard link: %v", path, err)
			return outcomeLinkFailure
		}
		return outcomeLinked
	}

	d.problem("%s: store entry kept changing; left untouched", path)
	return outcomeSkipped
}

// verify re-hashes a pre-existing store entry once per run before anything is
// linked to it, so a store file edited in place is never propagated further.
func (d *deduper) verify(entry string, sinfo fs.FileInfo, sum string, mode fs.FileMode) bool {
	if cached := d.verified[entry]; cached != nil && unchangedStoreFile(cached, sinfo) {
		return true
	}
	if !sinfo.Mode().IsRegular() || sinfo.Mode() != mode || storeEntryName(sum, sinfo) != filepath.Base(entry) {
		return false
	}
	got, err := hashFile(entry)
	if err != nil || got != sum {
		return false
	}
	after, err := os.Lstat(entry)
	if err != nil || !unchangedStoreFile(sinfo, after) {
		return false
	}
	d.verified[entry] = after
	return true
}

func unchangedStoreFile(before, after fs.FileInfo) bool {
	return after != nil && os.SameFile(before, after) && before.Size() == after.Size() &&
		before.ModTime().Equal(after.ModTime()) && before.Mode() == after.Mode() &&
		ownerTag(before) == ownerTag(after)
}

// replaceWithLink makes path a hard link to entry without ever leaving path
// missing: link to a temporary name in path's directory, then rename it over
// path (atomic on POSIX; MoveFileEx with MOVEFILE_REPLACE_EXISTING on Windows).
// If either step fails, path is untouched and the temporary name is removed.
func replaceWithLink(entry, path string) error {
	dir := filepath.Dir(path)
	for range 8 {
		tmp := filepath.Join(dir, dedupeTempPrefix+randomSuffix())
		if err := linkFile(entry, tmp); err != nil {
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return err
		}
		if err := renameFile(tmp, path); err != nil {
			_ = os.Remove(tmp)
			return err
		}
		return nil
	}
	return errors.New("could not pick an unused temporary name")
}

func randomSuffix() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func storeEntryName(sum string, info fs.FileInfo) string {
	name := fmt.Sprintf("%s-%o", sum, info.Mode().Perm())
	if tag := ownerTag(info); tag != "" {
		name += "-" + tag
	}
	return name
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := hashBufPool.Get().(*[]byte)
	defer hashBufPool.Put(buf)

	h := sha256.New()
	// Wrap f so CopyBuffer cannot take the WriterTo/ReaderFrom shortcut and
	// ignore buf.
	if _, err := io.CopyBuffer(h, struct{ io.Reader }{f}, *buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// hashBufPool avoids allocating (and zeroing) a fresh buffer for each of the
// ~18,000 files in a Zig install.
var hashBufPool = sync.Pool{New: func() any {
	b := make([]byte, 256<<10)
	return &b
}}

// diskUsage sums the sizes of regular files under every installed version and
// the store, counting each inode once — the space they actually occupy
// (allocation-block rounding aside).
func (z *ZVM) diskUsage() (int64, error) {
	installed, err := z.GetInstalledVersions()
	if err != nil {
		return 0, err
	}

	roots := make([]string, 0, len(installed)+1)
	for _, v := range installed {
		roots = append(roots, filepath.Join(z.baseDir, v))
	}
	roots = append(roots, z.storeDir())

	seen := make(map[fileID]struct{})
	var total int64
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) && path == root {
					return fs.SkipDir
				}
				return nil
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return nil
			}
			if id, _, ok := fileIdentity(path, info); ok {
				if _, dup := seen[id]; dup {
					return nil
				}
				seen[id] = struct{}{}
			}
			total += info.Size()
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}

// PruneStore garbage-collects the store: an entry whose link count is 1 is no
// longer referenced by any version directory and is removed. Entries whose
// link count cannot be read are kept.
func (z *ZVM) PruneStore() (PruneReport, error) {
	var report PruneReport
	store := z.storeDir()

	if _, err := os.Lstat(store); errors.Is(err, fs.ErrNotExist) {
		return report, nil
	}
	unlock, err := z.lockStore()
	if err != nil {
		return report, err
	}
	defer unlock()

	err = filepath.WalkDir(store, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			log.Debug("prune store", "path", path, "err", err)
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		// Fresh Lstat rather than a cached one: the link count must be current.
		info, err := os.Lstat(path)
		if err != nil {
			return nil
		}
		_, nlink, ok := fileIdentity(path, info)
		if !ok || nlink != 1 {
			return nil
		}
		if err := os.Remove(path); err != nil {
			log.Debug("prune store", "path", path, "err", err)
			return nil
		}
		report.Removed++
		report.BytesFreed += info.Size()
		return nil
	})
	if err != nil {
		return report, err
	}

	// Drop now-empty prefix directories; os.Remove refuses non-empty ones.
	if dirs, err := os.ReadDir(store); err == nil {
		for _, dir := range dirs {
			if dir.IsDir() {
				_ = os.Remove(filepath.Join(store, dir.Name()))
			}
		}
	}

	return report, nil
}

// autoDedupe runs after a successful install when Settings.AutoDedupe is set.
// It only processes the new version (whose files are matched against
// everything already in the store) and never fails the install.
func (z *ZVM) autoDedupe(version string) {
	report, err := z.Dedupe([]string{version})
	if err != nil {
		log.Warn("auto-dedupe failed; the install itself is fine", "err", err)
		return
	}
	if _, err := z.PruneStore(); err != nil {
		log.Debug("prune after auto-dedupe", "err", err)
	}
	fmt.Printf("Deduped %s: %d files shared with other versions, saved %s\n",
		version, report.Linked, FormatBytes(report.BytesSaved()))
}

// FormatBytes renders n with binary units (KiB, MiB, ...).
func FormatBytes(n int64) string {
	const unit = 1024
	if n < unit && n > -unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit || m <= -unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
