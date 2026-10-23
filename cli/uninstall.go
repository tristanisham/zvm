// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

func validateVersionTarget(version, action string) error {
	// Only accept a single install name, before shorthand normalization can
	// change dot-prefixed input. Reject both platforms' path separators.
	if version == "" || version == "." || version == ".." || strings.ContainsAny(version, `/\:`) {
		return fmt.Errorf("%w: %s requires a single installed version name", ErrInvalidInput, action)
	}
	return nil
}

func checkReservedVersion(version, action string) error {
	// These entries belong to ZVM itself, not to a Zig installation.
	if isReservedName(version) {
		return fmt.Errorf("%w: %w: cannot %s %q", ErrInvalidInput, ErrReservedName, action, version)
	}
	return nil
}

func (z *ZVM) isBinForVersion(version string, versionInfo os.FileInfo) bool {
	binPath := filepath.Join(z.baseDir, "bin")
	binLstat, err := os.Lstat(binPath)
	if err != nil || binLstat.IsDir() {
		return false
	}
	if binStat, err := os.Stat(binPath); err == nil && os.SameFile(versionInfo, binStat) {
		return true
	}
	if target, err := os.Readlink(binPath); err == nil {
		if !filepath.IsAbs(target) {
			target = filepath.Join(z.baseDir, target)
		}
		return filepath.Clean(target) == filepath.Clean(filepath.Join(z.baseDir, version))
	}
	return false
}

// Uninstall removes the specified Zig version from the ZVM base directory.
func (z *ZVM) Uninstall(version string) error {
	if err := validateVersionTarget(version, "uninstall"); err != nil {
		return err
	}
	// Resolve shorthand against locally installed versions
	if installedVersions, err := z.GetInstalledVersions(); err == nil {
		if resolved, resolveErr := resolveVersionShorthand(version, installedVersions); resolveErr == nil && resolved != version {
			log.Debug("resolved version shorthand", "input", version, "resolved", resolved)
			fmt.Printf("Resolved %q to %s\n", version, resolved)
			version = resolved
		}
	}

	if err := checkReservedVersion(version, "uninstall"); err != nil {
		return err
	}
	unlock, err := z.lockStore()
	if err != nil {
		return err
	}
	defer unlock()

	root, err := os.OpenRoot(z.baseDir)
	if err != nil {
		return err
	}
	defer root.Close()

	info, err := root.Lstat(version)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %q is a symlink, not a ZVM install; remove the link yourself if you no longer want it", ErrInvalidInput, version)
		}
		if !info.IsDir() {
			return fmt.Errorf("%w: uninstall target %q must be an install directory", ErrInvalidInput, version)
		}
		if !hasZigExecutable(root, version) {
			return fmt.Errorf("%w: %q is not a Zig installation (no zig executable inside)", ErrInvalidInput, version)
		}
		activeBin := z.isBinForVersion(version, info)
		// Deduped files are hard links: removing them only drops this
		// version's names. Other versions and the store keep their data.
		if err := root.RemoveAll(version); err != nil {
			return err
		}
		if activeBin {
			if err := os.Remove(filepath.Join(z.baseDir, "bin")); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		fmt.Printf("✔ Uninstalled %s\nRun `zvm ls` to view installed versions.\n", version)
		unlock()

		// Store entries only this version referenced are now unreferenced.
		if pruned, err := z.PruneStore(); err != nil {
			log.Warn("could not prune dedupe store", "err", err)
		} else if pruned.Removed > 0 {
			fmt.Printf("Freed %s from the dedupe store (%d files no longer shared).\n", FormatBytes(pruned.BytesFreed), pruned.Removed)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	fmt.Printf("Version: %s not found locally.\nHere are your installed versions:\n", version)
	return z.ListVersions()
}

// hasZigExecutable reports whether the install directory dir in root holds a
// zig or zig.exe file. Both names are accepted because --target-os can
// install a Windows build on any host.
func hasZigExecutable(root *os.Root, dir string) bool {
	for _, exe := range zigExeNames {
		if info, err := root.Lstat(filepath.Join(dir, exe)); err == nil && info.Mode().IsRegular() {
			return true
		}
	}
	return false
}
