// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
)

// Run the given Zig compiler with the provided arguments
func (z *ZVM) Run(version string, cmd []string) error {
	log.Debug("Run", "version", version, "cmds", strings.Join(cmd, ", "))
	if len(version) == 0 {
		return fmt.Errorf("no zig version provided. If you want to run your set version of Zig, please use 'zig'")
		// zig, err := z.zigPath()
		// log.Debug("Run", "zig path", zig)
		// if err != nil {
		// 	return fmt.Errorf("%w: no Zig version found; %w", ErrMissingBundlePath, err)
		// }

		// return z.runZig("bin", cmd)
	}

	installedVersions, err := z.GetInstalledVersions()
	if err != nil {
		return err
	}

	if slices.Contains(installedVersions, version) {
		return z.runZig(version, cmd)
	} else {
		rawVersionStructure, err := z.fetchVersionMap()
		if err != nil {
			return err
		}

		_, err = getTarPath(version, &rawVersionStructure)
		if err != nil {
			if errors.Is(err, ErrUnsupportedVersion) {
				return fmt.Errorf("%s: %q", err, version)
			} else {
				return err
			}
		}

		fmt.Printf("It looks like %s isn't installed. Would you like to install it? [y/n]\n", version)

		if getConfirmation() {
			if err = z.Install(version, false); err != nil {
				return err
			}
			return z.runZig(version, cmd)
		} else {
			return fmt.Errorf("version %s is not installed", version)
		}
	}

}

func (z *ZVM) runZig(version string, cmd []string) error {
	zigExe := "zig"
	if runtime.GOOS == "windows" {
		zigExe = "zig.exe"
	}

	bin := strings.TrimSpace(filepath.Join(z.Directories.state, version, zigExe))

	log.Debug("runZig", "bin", bin)
	if stat, err := os.Lstat(bin); err != nil {

		name := version
		if stat != nil {
			name = stat.Name()
		}
		return fmt.Errorf("%w: %s", err, name)
	}

	// the logging here really muddies up the output of the Zig compiler
	// and adds a lot of noise. For that reason this function exits with
	// the zig compilers exit code
	if err := execute(bin, cmd); err != nil {
		return err
	}

	return nil
}

// Execute the given Zig command with a specified compiler
func execute(bin string, cmd []string) error {
	// zvm run 0.14.0 build run --help
	if len(bin) == 0 {
		return fmt.Errorf("compiler binary cannot be empty")
	}

	zig := exec.Command(bin, cmd...)
	zig.Stdin, zig.Stdout, zig.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := zig.Run(); err != nil {
		if err2, ok := err.(*exec.ExitError); ok {
			os.Exit(err2.ExitCode())
		} else {
			return fmt.Errorf("error executing command '%s': %w", cmd, err2)
		}
	}

	return nil
}
