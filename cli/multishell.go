// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/tristanisham/zvm/cli/meta"
)

// Per-shell ("multishell") version switching.
//
// `zvm env` creates a link named <pid>_<unix-ms>_<random> inside
// MultishellDir(), exports its path as ZVM_MULTISHELL_PATH, and prepends it to
// PATH. While that variable names one of these links, `zvm use` repoints only
// that link, so other shells keep their own version.

const (
	// MultishellEnv names the per-shell link of the current shell.
	MultishellEnv = "ZVM_MULTISHELL_PATH"
	// MultishellDirEnv overrides where per-shell links are created.
	MultishellDirEnv = "ZVM_MULTISHELL_DIR"

	// MultishellMaxAge is how old a per-shell link must be before `zvm clean`
	// or `zvm env` may prune it. Links whose recorded shell pid is still alive
	// are kept regardless of age.
	MultishellMaxAge = 7 * 24 * time.Hour
)

// multishellNameRe matches the names of links created by NewMultishellLink.
// Pruning only ever considers entries whose name matches it.
var multishellNameRe = regexp.MustCompile(`^([0-9]+)_([0-9]+)_([0-9a-f]{8})$`)

// MultishellDir returns the directory holding per-shell links.
//
// It is deliberately outside the ZVM base directory, so `zvm ls` and
// GetInstalledVersions never mistake it for an installed Zig version, and
// outside os.TempDir, which is shared between users and periodically swept.
// In order of preference: $ZVM_MULTISHELL_DIR, $XDG_STATE_HOME/zvm/multishells,
// then <user cache dir>/zvm/multishells (~/.cache on Linux and BSD,
// ~/Library/Caches on macOS, %LocalAppData% on Windows).
func MultishellDir() (string, error) {
	if dir := os.Getenv(MultishellDirEnv); dir != "" {
		// A relative path would resolve differently after every cd.
		if !filepath.IsAbs(dir) {
			return "", fmt.Errorf("%w: %s must be an absolute path, got %q", ErrNoMultishellDir, MultishellDirEnv, dir)
		}
		return filepath.Clean(dir), nil
	}
	if state := os.Getenv("XDG_STATE_HOME"); state != "" && filepath.IsAbs(state) {
		return filepath.Join(state, "zvm", "multishells"), nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNoMultishellDir, err)
	}
	return filepath.Join(cache, "zvm", "multishells"), nil
}

// globalTarget returns what the global ~/.zvm/bin link points to. When there is
// no global version (or bin is not a link), it returns the bin path itself so a
// per-shell link created now follows whatever global version is set later.
func (z *ZVM) globalTarget() string {
	binDir := filepath.Join(z.baseDir, "bin")
	fi, err := os.Lstat(binDir)
	if err != nil || !meta.IsLink(fi) {
		return binDir
	}
	target, err := os.Readlink(binDir)
	if err != nil {
		return binDir
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(z.baseDir, target)
	}
	return target
}

// NewMultishellLink creates a fresh per-shell link pointing at the current
// global version and returns its path. pid is recorded in the name so pruning
// can tell whether the owning shell is still running.
func (z *ZVM) NewMultishellLink(pid int) (string, error) {
	dir, err := MultishellDir()
	if err != nil {
		return "", err
	}
	// Anything inside baseDir would show up in `zvm ls` as a Zig version.
	if rel, err := filepath.Rel(z.baseDir, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %s must not be inside %s", ErrNoMultishellDir, MultishellDirEnv, z.baseDir)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%d_%d_%s", pid, time.Now().UnixMilli(), hex.EncodeToString(suffix[:]))
	link := filepath.Join(dir, name)

	if err := meta.Link(z.globalTarget(), link); err != nil {
		return "", err
	}
	return link, nil
}

// ActiveMultishellLink returns the current shell's per-shell link, if
// ZVM_MULTISHELL_PATH names one. The link itself may not exist (for example it
// was pruned); it is recreated on the next switch. A set but unusable value is
// reported through err and must be treated as "no per-shell link".
func ActiveMultishellLink() (link string, ok bool, err error) {
	raw := os.Getenv(MultishellEnv)
	if raw == "" {
		return "", false, nil
	}

	dir, err := MultishellDir()
	if err != nil {
		return "", false, err
	}

	link = filepath.Clean(raw)
	if !filepath.IsAbs(link) || filepath.Dir(link) != filepath.Clean(dir) || !multishellNameRe.MatchString(filepath.Base(link)) {
		return "", false, fmt.Errorf("%w: %s=%q is not inside %q", ErrNotMultishellLink, MultishellEnv, raw, dir)
	}

	fi, err := os.Lstat(link)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return link, true, nil
	case err != nil:
		return "", false, err
	case !meta.IsLink(fi):
		return "", false, fmt.Errorf("%w: %q is not a link", ErrNotMultishellLink, link)
	}
	return link, true, nil
}

// setShellLink repoints a per-shell link at an installed version.
func (z *ZVM) setShellLink(link, ver string) error {
	// The link sits at the front of PATH, so it must only ever point at an
	// install directory inside baseDir.
	if _, err := z.zigExecutable(ver); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(link), 0o700); err != nil {
		return err
	}
	if err := meta.Relink(filepath.Join(z.baseDir, ver), link); err != nil {
		return err
	}
	log.Debug("Use (this shell)", "version", ver, "link", link)
	return nil
}

// linkPointsTo reports whether link currently points at target. It reads the
// link without following it.
func linkPointsTo(link, target string) bool {
	got, err := os.Readlink(link)
	if err != nil {
		return false
	}
	return filepath.Clean(got) == filepath.Clean(target)
}

// PruneMultishells removes per-shell links that ZVM created and that are older
// than maxAge, unless the shell pid recorded in the name is still running. It
// only removes entries whose name matches the ZVM naming scheme and which are
// links; it never follows them, so nothing a link points to is touched. It
// returns the number of links removed.
func PruneMultishells(maxAge time.Duration) (int, error) {
	dir, err := MultishellDir()
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	current := filepath.Clean(os.Getenv(MultishellEnv))
	removed := 0
	var errs error
	for _, entry := range entries {
		m := multishellNameRe.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if path == current {
			continue
		}

		fi, err := os.Lstat(path)
		if err != nil || !meta.IsLink(fi) {
			continue
		}
		if time.Since(fi.ModTime()) < maxAge {
			continue
		}
		if pid, err := strconv.Atoi(m[1]); err == nil {
			if alive, known := processAlive(pid); known && alive {
				continue
			}
		}

		if err := os.Remove(path); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		log.Debug("pruned per-shell link", "path", path)
		removed++
	}
	return removed, errs
}
