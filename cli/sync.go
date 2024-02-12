// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (z *ZVM) Sync() error {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	buildZigPath := filepath.Join(cwd, "build.zig")
	if _, err := os.Stat(buildZigPath); err != nil {
		return fmt.Errorf("build.zig not found in %q", buildZigPath)
	}

	buildFile, err := os.Open(buildZigPath)
	if err != nil {
		return fmt.Errorf("error opening build.zig: %q", err)
	}
	defer buildFile.Close()

	scanner := bufio.NewScanner(buildFile)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(strings.TrimSpace(line), "//!") {
			continue
		}

		variable := line[3:]
		variablePieces := strings.Split(strings.TrimSpace(variable), ":")
		if len(variablePieces) > 2 {
			return fmt.Errorf("improper config variable formatting: %q. Should be key: value", variablePieces)
		}

		switch strings.ToLower(strings.TrimSpace(variablePieces[0])) {
		case "zvm-lock":
			val := strings.TrimSpace(variablePieces[1])
			if err := z.Use(val); err != nil {
				return err
			}
		}
	}

	return nil
}
