// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"archive/zip"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
)

type storeTestFile struct {
	path    string // "<version>/<rel path>"
	content string
	mode    fs.FileMode
	symlink string // when set, path is a symlink to this target
}

func newStoreTestZVM(t *testing.T, files []storeTestFile) *ZVM {
	t.Helper()
	z := &ZVM{baseDir: t.TempDir()}
	for _, f := range files {
		writeStoreTestFile(t, z, f)
	}
	return z
}

func writeStoreTestFile(t *testing.T, z *ZVM, f storeTestFile) {
	t.Helper()
	full := filepath.Join(z.baseDir, filepath.FromSlash(f.path))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if f.symlink != "" {
		if err := os.Symlink(f.symlink, full); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		return
	}
	mode := f.mode
	if mode == 0 {
		mode = 0644
	}
	if err := os.WriteFile(full, []byte(f.content), mode); err != nil {
		t.Fatal(err)
	}
	// WriteFile is subject to the umask; set the exact mode under test.
	if err := os.Chmod(full, mode); err != nil {
		t.Fatal(err)
	}
}

func (z *ZVM) testPath(rel string) string {
	return filepath.Join(z.baseDir, filepath.FromSlash(rel))
}

func sameFile(t *testing.T, a, b string) bool {
	t.Helper()
	ai, err := os.Lstat(a)
	if err != nil {
		t.Fatal(err)
	}
	bi, err := os.Lstat(b)
	if err != nil {
		t.Fatal(err)
	}
	return os.SameFile(ai, bi)
}

// assertUnchanged checks every non-symlink file still has its content and mode.
func assertUnchanged(t *testing.T, z *ZVM, files []storeTestFile) {
	t.Helper()
	for _, f := range files {
		full := z.testPath(f.path)
		info, err := os.Lstat(full)
		if err != nil {
			t.Fatalf("%s missing after dedupe: %v", f.path, err)
		}
		if f.symlink != "" {
			if info.Mode()&fs.ModeSymlink == 0 {
				t.Fatalf("%s: symlink was replaced", f.path)
			}
			continue
		}
		got, err := os.ReadFile(full)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != f.content {
			t.Fatalf("%s content = %q, want %q", f.path, got, f.content)
		}
		want := f.mode
		if want == 0 {
			want = 0644
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != want {
			t.Fatalf("%s mode = %v, want %v", f.path, info.Mode().Perm(), want)
		}
	}
}

func storeEntries(t *testing.T, z *ZVM) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(z.storeDir(), func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			rel, _ := filepath.Rel(z.storeDir(), path)
			out = append(out, rel)
		}
		return nil
	})
	return out
}

func TestDedupe(t *testing.T) {
	const big = "pub fn main() void {}\n"

	tests := []struct {
		name        string
		files       []storeTestFile
		same        [][2]string // pairs that must end up as one inode
		different   [][2]string // pairs that must stay separate inodes
		wantLinked  int
		wantStored  int
		wantIgnored int
		wantSaved   int64
		unixOnly    bool
	}{
		{
			name: "identical files across versions are merged",
			files: []storeTestFile{
				{path: "0.15.1/lib/std/std.zig", content: big},
				{path: "0.15.2/lib/std/std.zig", content: big},
			},
			same:       [][2]string{{"0.15.1/lib/std/std.zig", "0.15.2/lib/std/std.zig"}},
			wantStored: 1,
			wantLinked: 1,
			wantSaved:  int64(len(big)),
		},
		{
			name: "differing files are kept",
			files: []storeTestFile{
				{path: "0.15.1/zig", content: "binary one", mode: 0755},
				{path: "0.15.2/zig", content: "binary two", mode: 0755},
			},
			different:  [][2]string{{"0.15.1/zig", "0.15.2/zig"}},
			wantStored: 2,
		},
		{
			name: "same content with different modes stays separate",
			files: []storeTestFile{
				{path: "0.15.1/tool", content: big, mode: 0644},
				{path: "0.15.2/tool", content: big, mode: 0755},
				{path: "0.15.3/tool", content: big, mode: 0755},
			},
			different:  [][2]string{{"0.15.1/tool", "0.15.2/tool"}},
			same:       [][2]string{{"0.15.2/tool", "0.15.3/tool"}},
			wantStored: 2,
			wantLinked: 1,
			wantSaved:  int64(len(big)),
			unixOnly:   true, // Windows has no 0644/0755 distinction
		},
		{
			name: "symlinks and empty files are left alone",
			files: []storeTestFile{
				{path: "0.15.1/lib/a.zig", content: big},
				{path: "0.15.1/lib/link.zig", symlink: "a.zig"},
				{path: "0.15.2/lib/link.zig", symlink: "a.zig"},
				{path: "0.15.1/empty", content: ""},
				{path: "0.15.2/empty", content: ""},
			},
			different:   [][2]string{{"0.15.1/empty", "0.15.2/empty"}},
			wantStored:  1,
			wantIgnored: 2,
		},
		{
			name: "duplicates within one version are merged too",
			files: []storeTestFile{
				{path: "0.15.1/lib/LICENSE", content: big},
				{path: "0.15.1/lib/sub/LICENSE", content: big},
			},
			same:       [][2]string{{"0.15.1/lib/LICENSE", "0.15.1/lib/sub/LICENSE"}},
			wantStored: 1,
			wantLinked: 1,
			wantSaved:  int64(len(big)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unixOnly && runtime.GOOS == "windows" {
				t.Skip("unix permissions")
			}
			z := newStoreTestZVM(t, tt.files)

			report, err := z.Dedupe(nil)
			if err != nil {
				t.Fatalf("Dedupe: %v", err)
			}

			if report.Linked != tt.wantLinked || report.Stored != tt.wantStored || report.Ignored != tt.wantIgnored || report.Skipped != 0 {
				t.Fatalf("report = %+v; want linked=%d stored=%d ignored=%d skipped=0",
					report, tt.wantLinked, tt.wantStored, tt.wantIgnored)
			}
			if report.BytesSaved() != tt.wantSaved {
				t.Fatalf("BytesSaved = %d, want %d (before %d, after %d)", report.BytesSaved(), tt.wantSaved, report.BytesBefore, report.BytesAfter)
			}
			for _, p := range tt.same {
				if !sameFile(t, z.testPath(p[0]), z.testPath(p[1])) {
					t.Errorf("%s and %s should be one inode", p[0], p[1])
				}
			}
			for _, p := range tt.different {
				if sameFile(t, z.testPath(p[0]), z.testPath(p[1])) {
					t.Errorf("%s and %s must not be merged", p[0], p[1])
				}
			}
			assertUnchanged(t, z, tt.files)
		})
	}
}

type inodeSnapshot map[string]fileID

func snapshot(t *testing.T, z *ZVM) inodeSnapshot {
	t.Helper()
	snap := inodeSnapshot{}
	_ = filepath.WalkDir(z.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			t.Fatal(err)
		}
		id, _, ok := fileIdentity(path, info)
		if !ok {
			t.Skip("file identity unavailable on this platform")
		}
		snap[path] = id
		return nil
	})
	return snap
}

func TestDedupeIsIdempotent(t *testing.T) {
	files := []storeTestFile{
		{path: "0.15.1/lib/std.zig", content: "shared"},
		{path: "0.15.2/lib/std.zig", content: "shared"},
		{path: "0.15.1/zig", content: "one", mode: 0755},
		{path: "0.15.2/zig", content: "two", mode: 0755},
	}
	z := newStoreTestZVM(t, files)

	if _, err := z.Dedupe(nil); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, z)

	second, err := z.Dedupe(nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.Stored != 0 || second.Linked != 0 || second.Skipped != 0 || second.BytesSaved() != 0 {
		t.Fatalf("second run changed something: %+v", second)
	}
	if second.AlreadyLinked != len(files) {
		t.Fatalf("AlreadyLinked = %d, want %d", second.AlreadyLinked, len(files))
	}
	after := snapshot(t, z)
	if len(before) != len(after) {
		t.Fatalf("file set changed: %d -> %d", len(before), len(after))
	}
	for path, id := range before {
		if after[path] != id {
			t.Errorf("%s was relinked on the second run", path)
		}
	}
	assertUnchanged(t, z, files)
}

func TestDedupeLinkFailuresLeaveFilesAlone(t *testing.T) {
	tests := []struct {
		name string
		err  error
		// failOnlyReplace fails only the temp link made while replacing a
		// file, i.e. after the store entry already exists.
		failOnlyReplace bool
		files           int
		wantSkipped     int
		wantGiveUp      bool
	}{
		{name: "cross-device (EXDEV)", err: syscall.EXDEV, files: 3, wantSkipped: 3},
		{name: "no hard links (EPERM, e.g. FAT)", err: syscall.EPERM, files: 3, wantSkipped: 3},
		{name: "gives up after repeated failures", err: syscall.EXDEV, files: maxConsecutiveLinkFailures + 5, wantSkipped: maxConsecutiveLinkFailures, wantGiveUp: true},
		{name: "replace step fails", err: syscall.EMLINK, failOnlyReplace: true, files: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var files []storeTestFile
			for i := range tt.files {
				name := "lib/f" + string(rune('a'+i))
				files = append(files,
					storeTestFile{path: "0.15.1/" + name, content: "content " + name},
					storeTestFile{path: "0.15.2/" + name, content: "content " + name})
			}
			z := newStoreTestZVM(t, files)
			before := snapshot(t, z)

			linkFile = func(oldname, newname string) error {
				if tt.failOnlyReplace && !strings.HasPrefix(filepath.Base(newname), dedupeTempPrefix) {
					return os.Link(oldname, newname)
				}
				return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: tt.err}
			}
			t.Cleanup(func() { linkFile = os.Link })

			report, err := z.Dedupe([]string{"0.15.1", "0.15.2"})
			if err != nil {
				t.Fatalf("a link failure must not fail the run: %v", err)
			}

			wantSkipped := tt.wantSkipped
			if tt.failOnlyReplace {
				// 0.15.1 seeds the store fine; every 0.15.2 replace fails.
				wantSkipped = tt.files
			} else {
				wantSkipped *= 2 // both versions
			}
			if report.Skipped != wantSkipped {
				t.Fatalf("Skipped = %d, want %d (%+v)", report.Skipped, wantSkipped, report)
			}
			if report.Linked != 0 {
				t.Fatalf("Linked = %d, want 0", report.Linked)
			}
			gaveUp := slices.ContainsFunc(report.Problems, func(p string) bool { return strings.Contains(p, "in a row failed") })
			if gaveUp != tt.wantGiveUp {
				t.Fatalf("gave up = %v, want %v; problems: %v", gaveUp, tt.wantGiveUp, report.Problems)
			}

			// No version file was replaced, and no temp link was left behind.
			after := snapshot(t, z)
			for path, id := range before {
				if after[path] != id {
					t.Errorf("%s was replaced despite link failure", path)
				}
				delete(after, path)
			}
			for path := range after {
				if !strings.HasPrefix(path, z.storeDir()) {
					t.Errorf("unexpected leftover %s", path)
				}
			}
			assertUnchanged(t, z, files)
		})
	}
}

func TestDedupeInterruptedReplace(t *testing.T) {
	files := []storeTestFile{
		{path: "0.15.1/lib/std.zig", content: "shared"},
		{path: "0.15.2/lib/std.zig", content: "shared"},
	}
	z := newStoreTestZVM(t, files)

	// A rename that fails (as if interrupted between link and rename) must
	// leave the original in place and clean up its temporary link.
	renameFile = func(string, string) error { return syscall.EIO }
	report, err := z.Dedupe(nil)
	renameFile = os.Rename
	if err != nil {
		t.Fatal(err)
	}
	if report.Linked != 0 || report.Skipped != 1 {
		t.Fatalf("report = %+v", report)
	}
	assertUnchanged(t, z, files)
	tmps, _ := filepath.Glob(filepath.Join(z.baseDir, "0.15.2", "lib", dedupeTempPrefix+"*"))
	if len(tmps) != 0 {
		t.Fatalf("temp links left behind: %v", tmps)
	}

	// A process killed after the link but before the rename leaves a temp
	// link; the next run removes it and completes the dedupe.
	leftover := filepath.Join(z.baseDir, "0.15.2", "lib", dedupeTempPrefix+"deadbeef")
	if err := os.Link(z.testPath("0.15.1/lib/std.zig"), leftover); err != nil {
		t.Fatal(err)
	}
	report, err = z.Dedupe(nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Linked != 1 {
		t.Fatalf("report = %+v", report)
	}
	if _, err := os.Lstat(leftover); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("leftover temp link not removed: %v", err)
	}
	assertUnchanged(t, z, files)
}

func TestDedupeDetectsStoreEntryModifiedInPlace(t *testing.T) {
	files := []storeTestFile{
		{path: "0.15.1/lib/std.zig", content: "original"},
		{path: "0.15.2/lib/std.zig", content: "original"},
	}
	z := newStoreTestZVM(t, files)
	if _, err := z.Dedupe(nil); err != nil {
		t.Fatal(err)
	}

	// Editing a deduped file in place edits every version sharing it — the
	// documented trade-off of hard links.
	if err := os.WriteFile(z.testPath("0.15.1/lib/std.zig"), []byte("edited"), 0644); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(z.testPath("0.15.2/lib/std.zig")); string(got) != "edited" {
		t.Fatalf("expected the edit to be shared, got %q", got)
	}

	// A new version with the original content must not be linked to the
	// stale entry (whose name still claims "original").
	fresh := storeTestFile{path: "0.15.3/lib/std.zig", content: "original"}
	writeStoreTestFile(t, z, fresh)
	report, err := z.Dedupe(nil)
	if err != nil {
		t.Fatal(err)
	}
	assertUnchanged(t, z, []storeTestFile{fresh})
	if !slices.ContainsFunc(report.Problems, func(p string) bool { return strings.Contains(p, "modified in place") }) {
		t.Fatalf("expected a warning about the modified entry, got %v", report.Problems)
	}
	if sameFile(t, z.testPath("0.15.2/lib/std.zig"), z.testPath(fresh.path)) {
		t.Fatal("fresh file was linked to edited content")
	}
}

func TestPruneStoreRemovesOnlyUnreferencedEntries(t *testing.T) {
	files := []storeTestFile{
		{path: "0.15.1/lib/shared.zig", content: "shared"},
		{path: "0.15.2/lib/shared.zig", content: "shared"},
		{path: "0.15.1/zig", content: "only in 0.15.1", mode: 0755},
		{path: "0.15.2/zig", content: "only in 0.15.2", mode: 0755},
	}
	z := newStoreTestZVM(t, files)
	if _, err := z.Dedupe(nil); err != nil {
		t.Fatal(err)
	}
	if n := len(storeEntries(t, z)); n != 3 {
		t.Fatalf("store has %d entries, want 3", n)
	}

	// Nothing is unreferenced yet.
	pruned, err := z.PruneStore()
	if err != nil {
		t.Fatal(err)
	}
	if pruned.Removed != 0 {
		t.Fatalf("pruned %d referenced entries", pruned.Removed)
	}

	// Uninstall prunes what only 0.15.1 referenced and nothing else.
	if err := z.Uninstall("0.15.1"); err != nil {
		t.Fatal(err)
	}
	if n := len(storeEntries(t, z)); n != 2 {
		t.Fatalf("store has %d entries after uninstall, want 2", n)
	}
	assertUnchanged(t, z, []storeTestFile{files[1], files[3]})

	// A version removed behind zvm's back is collected by PruneStore (clean).
	if err := os.RemoveAll(filepath.Join(z.baseDir, "0.15.2")); err != nil {
		t.Fatal(err)
	}
	pruned, err = z.PruneStore()
	if err != nil {
		t.Fatal(err)
	}
	if pruned.Removed != 2 || pruned.BytesFreed != int64(len("shared")+len("only in 0.15.2")) {
		t.Fatalf("pruned = %+v", pruned)
	}
	if n := len(storeEntries(t, z)); n != 0 {
		t.Fatalf("store not empty: %v", storeEntries(t, z))
	}
}

func TestReservedDirsAreNotVersions(t *testing.T) {
	z := newStoreTestZVM(t, []storeTestFile{
		{path: "0.15.1/zig", content: "zig", mode: 0755},
		{path: "0.15.2/zig", content: "zig", mode: 0755},
		{path: "self/zvm", content: "zvm", mode: 0755},
	})
	if err := os.Symlink(filepath.Join(z.baseDir, "0.15.1"), filepath.Join(z.baseDir, "bin")); err != nil {
		if err := os.Mkdir(filepath.Join(z.baseDir, "bin"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := z.Dedupe(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(z.storeDir()); err != nil {
		t.Fatalf("store dir not created: %v", err)
	}

	got, err := z.GetInstalledVersions()
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if !slices.Equal(got, []string{"0.15.1", "0.15.2"}) {
		t.Fatalf("GetInstalledVersions = %v; store/bin/self must not be listed", got)
	}

	ctx := context.Background()
	for _, name := range []string{"store", "STORE", "bin", "self"} {
		tests := []struct {
			op      string
			run     func() error
			wantErr error
		}{
			{"uninstall", func() error { return z.Uninstall(name) }, ErrReservedName},
			{"use", func() error { _, err := z.Use(name); return err }, ErrReservedName},
			{"install", func() error { _, err := z.Install(name, true, true, false, true); return err }, ErrReservedName},
			{"install zls", func() error { return z.InstallZls(name, "only-runtime", true, true, true) }, ErrReservedName},
			{"dedupe", func() error { _, err := z.Dedupe([]string{name}); return err }, ErrReservedName},
			{"alias", func() error { v := name; return z.Alias(ctx, "x", &v) }, ErrInvalidAliasValue},
		}
		for _, tt := range tests {
			t.Run(tt.op+" "+name, func(t *testing.T) {
				err := tt.run()
				// uninstall and use reject bin and self in
				// checkReservedVersion (ErrInvalidInput) before
				// isReservedName is reached.
				rejectedEarlier := tt.wantErr == ErrReservedName && errors.Is(err, ErrInvalidInput)
				if !errors.Is(err, tt.wantErr) && !rejectedEarlier {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				if z.IsInstalled(name) {
					t.Fatalf("IsInstalled(%q) = true", name)
				}
			})
		}
	}

	if _, err := os.Stat(z.storeDir()); err != nil {
		t.Fatalf("store dir was removed: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(z.baseDir, "bin")); err != nil {
		t.Fatalf("bin was removed: %v", err)
	}
	if entries := storeEntries(t, z); len(entries) != 1 {
		t.Fatalf("store entries = %v, want the one shared zig binary", entries)
	}
}

func TestDedupeRejectsUnknownVersion(t *testing.T) {
	z := newStoreTestZVM(t, []storeTestFile{{path: "0.15.1/zig", content: "zig"}})
	if _, err := z.Dedupe([]string{"0.99.0"}); !errors.Is(err, ErrVersionNotInstalled) {
		t.Fatalf("err = %v, want ErrVersionNotInstalled", err)
	}
	// Shorthand resolves against installed versions.
	report, err := z.Dedupe([]string{"0.15"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(report.Versions, []string{"0.15.1"}) {
		t.Fatalf("Versions = %v", report.Versions)
	}
}

// Re-extracting over a deduped file must replace it, not write through the
// hard link into the store and every other version.
func TestExtractionDoesNotWriteThroughHardLinks(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "other-version-file")
	if err := os.WriteFile(other, []byte("shared"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(other, filepath.Join(dest, "std.zig")); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}

	archive := filepath.Join(dir, "bundle.zip")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	entry, err := w.Create("std.zig")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("new content")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	if err := unzipSource(archive, dest); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dest, "std.zig")); string(got) != "new content" {
		t.Fatalf("extracted = %q", got)
	}
	if got, _ := os.ReadFile(other); string(got) != "shared" {
		t.Fatalf("extraction wrote through the hard link: other file = %q", got)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{176 * 1024 * 1024, "176.0 MiB"},
		{3 << 30, "3.0 GiB"},
	}
	for _, tt := range tests {
		if got := FormatBytes(tt.in); got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Dedupe must never hard-link files inside a directory a version symlink
// points to: that is the user's own build tree, not ZVM's.
func TestDedupeSkipsSymlinkedVersion(t *testing.T) {
	z := newStoreTestZVM(t, []storeTestFile{{path: "0.15.1/lib/std.zig", content: "same", mode: 0o755}, {path: "0.15.1/zig", content: "zig", mode: 0o755}})
	build := t.TempDir()
	for name, data := range map[string]string{"lib/std.zig": "same", "zig": "zig"} {
		path := filepath.Join(build, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(build, filepath.Join(z.baseDir, "local")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := z.Dedupe(nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"lib/std.zig", "zig"} {
		info, err := os.Stat(filepath.Join(build, name))
		if err != nil {
			t.Fatal(err)
		}
		if _, n, ok := fileIdentity(filepath.Join(build, name), info); ok && n != 1 {
			t.Errorf("%s in the local build has %d links; dedupe must not touch it", name, n)
		}
	}
}
