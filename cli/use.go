// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	// "github.com/charmbracelet/log"
	"github.com/tristanisham/zvm/cli/meta"
)

func (z *ZVM) Use(ver string) error {
	if err := z.getVersion(ver); err != nil {
		if errors.Is(err, os.ErrNotExist) {

			fmt.Printf("It looks like %s isn't installed. Would you like to install it? [y/n]\n", ver)
			if getConfirmation() {
				if err = z.Install(ver); err != nil {
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

	// Remove "bin" dir only if it already exists
	// According to https://stackoverflow.com/a/12518877/598919
	// errors.Is(err, os.ErrNotExist) should be used
	if _, err := os.Stat(bin_dir); !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Removing old %s", bin_dir)
		if err := os.Remove(bin_dir); err != nil {
			return err
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
