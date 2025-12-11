// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"os"
)

// Uninstall removes the specified Zig version from the ZVM base directory.
func (z *ZVM) Uninstall(version string) error {
	root, err := os.OpenRoot(z.baseDir)
	if err != nil {
		return err
	}

	if _, err := root.Stat(version); err == nil {
		if err := root.RemoveAll(version); err != nil {
			return err
		}
		fmt.Printf("✔ Uninstalled %s.\nRun `zvm ls` to view installed versions.\n", version)
		return nil
	}
	fmt.Printf("Version: %s not found locally.\nHere are your installed versions:\n", version)
	return z.ListVersions()
}
