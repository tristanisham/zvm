// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package cli

import (
	"os"
	"path/filepath"
)

// Clean removes any compressed archives (.zip, .xz, .tar) from the ZVM base directory
// to save disk space. It leaves the installed version directories intact.
func (z *ZVM) Clean() error {
	dir, err := os.ReadDir(z.baseDir)
	if err != nil {
		return err
	}

	for _, entry := range dir {
		if filepath.Ext(entry.Name()) == ".zip" || filepath.Ext(entry.Name()) == ".xz" || filepath.Ext(entry.Name()) == ".tar" {
			if err := os.Remove(filepath.Join(z.baseDir, entry.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}
