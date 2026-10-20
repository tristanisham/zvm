// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
)

// Uninstall removes the specified Zig version from the ZVM base directory.
func (z *ZVM) Uninstall(version string) error {
	// Resolve shorthand against locally installed versions
	if installedVersions, err := z.GetInstalledVersions(); err == nil {
		if resolved, resolveErr := resolveVersionShorthand(version, installedVersions); resolveErr == nil && resolved != version {
			log.Debug("resolved version shorthand", "input", version, "resolved", resolved)
			fmt.Printf("Resolved %q to %s\n", version, resolved)
			version = resolved
		}
	}

	// Never remove ZVM's own directories (bin, self, the dedupe store), even
	// though they sit next to the versions in the base directory.
	if isReservedName(version) {
		return fmt.Errorf("%w: %q", ErrReservedName, version)
	}

	root, err := os.OpenRoot(z.baseDir)
	if err != nil {
		return err
	}
	defer root.Close()

	if _, err := root.Stat(version); err == nil {
		// Deduped files are hard links: removing them only drops this
		// version's names. Other versions and the store keep their data.
		if err := root.RemoveAll(version); err != nil {
			return err
		}
		fmt.Printf("✔ Uninstalled %s\nRun `zvm ls` to view installed versions.\n", version)

		// Store entries only this version referenced are now unreferenced.
		if pruned, err := z.PruneStore(); err != nil {
			log.Warn("could not prune dedupe store", "err", err)
		} else if pruned.Removed > 0 {
			fmt.Printf("Freed %s from the dedupe store (%d files no longer shared).\n", FormatBytes(pruned.BytesFreed), pruned.Removed)
		}
		return nil
	}
	fmt.Printf("Version: %s not found locally.\nHere are your installed versions:\n", version)
	return z.ListVersions()
}
