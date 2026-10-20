// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

// Uninstall removes the specified Zig version from the ZVM base directory.
func (z *ZVM) Uninstall(version string) error {
	// Only accept a single install name, before shorthand normalization can
	// change dot-prefixed input. Reject both platforms' path separators.
	if version == "" || version == "." || version == ".." || strings.ContainsAny(version, `/\:`) {
		return fmt.Errorf("%w: uninstall requires a single installed version name", ErrInvalidInput)
	}
	// Resolve shorthand against locally installed versions
	if installedVersions, err := z.GetInstalledVersions(); err == nil {
		if resolved, resolveErr := resolveVersionShorthand(version, installedVersions); resolveErr == nil && resolved != version {
			log.Debug("resolved version shorthand", "input", version, "resolved", resolved)
			fmt.Printf("Resolved %q to %s\n", version, resolved)
			version = resolved
		}
	}

	// These entries belong to ZVM itself, not to a Zig installation.
	switch strings.ToLower(version) {
	case "settings.json", "self", "bin":
		return fmt.Errorf("%w: cannot uninstall reserved entry %q", ErrInvalidInput, version)
	}

	root, err := os.OpenRoot(z.baseDir)
	if err != nil {
		return err
	}
	defer root.Close()

	info, err := root.Lstat(version)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%w: uninstall target %q must be an install directory", ErrInvalidInput, version)
		}
		if err := root.RemoveAll(version); err != nil {
			return err
		}
		fmt.Printf("✔ Uninstalled %s\nRun `zvm ls` to view installed versions.\n", version)
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	fmt.Printf("Version: %s not found locally.\nHere are your installed versions:\n", version)
	return z.ListVersions()
}
