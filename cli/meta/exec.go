// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Execute the given Zig command with a specified compiler
func Exec(bin string, cmd []string) error {
	// zvm run 0.14.0 build run --help
	if bin == "" {
		return fmt.Errorf("compiler binary cannot be empty")
	}

	zig := exec.Command(bin, cmd...)
	zig.Stdin, zig.Stdout, zig.Stderr = os.Stdin, os.Stdout, os.Stderr

	err := zig.Run()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			os.Exit(err.ExitCode())
		} else {
			return fmt.Errorf("Error executing command '%s': %s\n", cmd, err)
		}
	}

	return nil
}

// Execute the given Zig command with a specified compiler
// This is a convenience function that will automatically split a
// command and execute it
func ExecString(cmd string) error {
	command := strings.Split(cmd, " ")
	if len(command) < 1 {
		return fmt.Errorf("No command given")
	}

	zig := exec.Command(command[0], command[1:]...)
	zig.Stdin, zig.Stdout, zig.Stderr = os.Stdin, os.Stdout, os.Stderr

	err := zig.Run()
	if err != nil {
		return fmt.Errorf("Error executing command '%s': %s\n", cmd, err)
	}

	return nil
}
