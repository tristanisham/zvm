// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
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

	z.createSymlinks(ver)
	return nil
}

func getConfirmation() bool {
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	answer := strings.TrimSpace(strings.ToLower(text))
	return answer == "y" || answer == "ye" || answer == "yes"
}
