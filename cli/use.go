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
)

func (z *ZVM) Use(ver string) error {
	err := z.getVersion(ver)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Try running 'zvm i %s\n", ver)
		return fmt.Errorf("version %s is not installed", ver)
	}

	if err != nil {
		return err
	}

	return z.setBin(ver)
}

func (z *ZVM) setBin(ver string) error {
	version_path := filepath.Join(z.baseDir, ver)
	if err := os.Remove(filepath.Join(z.baseDir, "bin")); err != nil {
		log.Warn(err)
	}

	if err := os.Symlink(filepath.Join(z.baseDir, ver), filepath.Join(z.baseDir, "bin")); err != nil {
		log.Fatal(err)
	}

	if err := os.Symlink(version_path, filepath.Join(z.baseDir, "bin")); err != nil {
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
