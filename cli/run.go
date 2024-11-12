// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tristanisham/zvm/cli/meta"
)

// Run the given Zig compiler with the provided arguments
func (z *ZVM) Run(ver string, cmd []string) error {
	if err := z.getVersion(ver); err != nil {
		if errors.Is(err, os.ErrNotExist) {

			fmt.Printf("It looks like %s isn't installed. Would you like to install it first? [y/n]\n", ver)

			if getConfirmation() {
				if err = z.Install(ver, false); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("version %s is not installed", ver)
			}
		}
	}

	return z.runBin(ver, cmd)
}

func (z *ZVM) runBin(ver string, cmd []string) error {
	// $ZVM_PATH/$VERSION/zig cmd
	bin := filepath.Join(z.baseDir, ver, "zig")

	// Skip symlink checks, does this Zig binary exist?
	stat, err := os.Stat(bin)
	if err != nil {
		return fmt.Errorf("%w: %s", err, stat.Name())
	}

	// the logging here really muddies up the output of the Zig compiler
	// and adds a lot of noise. For that reason this function exits with
	// the zig compilers exit code
	if err := meta.Exec(bin, cmd); err != nil {
		return err
	}

	return nil
}
