// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

func (z *ZVM) Uninstall(ver string) error {
	version := filepath.Join(z.baseDir, ver)

	if _, err := os.Stat(version); err == nil {
		if err := os.RemoveAll(version); err != nil {
			return err
		}
		fmt.Printf("✔ Uninstalled %s.\nRun `zvm ls` to view installed versions.\n", ver)
		return nil
	}
	fmt.Printf("Version: %s not found locally.\nHere are your installed versions:\n", ver)
	return z.ListVersions()
}
