// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/tristanisham/zvm/cli/meta"
)

func (z *ZVM) Use(ver string) error {
	if err := z.getVersion(ver); err != nil {
		if errors.Is(err, os.ErrNotExist) {

			fmt.Printf("It looks like %s isn't installed. Would you like to install it? [y/n]\n", ver)
			if getConfirmation() {
				if err = z.Install(ver, false); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("version %s is not installed", ver)
			}
		}
	}

	return z.setBin(ver)
}

func (z *ZVM) setBin(ver string) error {
	// .zvm/master
	version_path := filepath.Join(z.baseDir, ver)
	bin_dir := filepath.Join(z.baseDir, "bin")

	// Came across https://pkg.go.dev/os#Lstat
	// which is specifically to check symbolic links.
	// Seemed like the more appropriate solution here
	stat, err := os.Lstat(bin_dir)

	// Actually we need to check if the symbolic link to ~/.zvm/bin
	// exists yet, otherwise we get err:
	//
	// CreateFile C:\Users\gs\.zvm\bin: The system cannot find the file specified.
	//
	// which leads to evaluation of the else case (line 59) and to an early return
	// therefore the the initial symbolic link is never created.
	if stat != nil {
		if err == nil {
			log.Debugf("Removing old %s", bin_dir)
			if err := os.Remove(bin_dir); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%w: %s", err, stat.Name())
		}
	}

	if err := meta.Symlink(version_path, bin_dir); err != nil {
		return err
	}

	return nil
}

func getConfirmation() bool {
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	answer := strings.TrimSpace(strings.ToLower(text))
	return answer == "y" || answer == "ye" || answer == "yes"
}
