// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"strings"

	"github.com/charmbracelet/log"
	"github.com/tristanisham/zvm/cli/meta"
)

// Use switches the active Zig version to the specified one.
// If the version is not installed, it prompts the user to install it.
// Returns the resolved version string (which may differ from input if shorthand was used).
func (z *ZVM) Use(ver string) (string, error) {
	// Resolve shorthand against locally installed versions
	if installedVersions, err := z.GetInstalledVersions(); err == nil {
		if resolved, resolveErr := resolveVersionShorthand(ver, installedVersions); resolveErr == nil && resolved != ver {
			log.Debug("resolved version shorthand", "input", ver, "resolved", resolved)
			fmt.Printf("Resolved %q to %s\n", ver, resolved)
			ver = resolved
		}
	}

	if err := z.getVersion(ver); err != nil {
		if errors.Is(err, os.ErrNotExist) {

			fmt.Printf("It looks like %s isn't installed. Would you like to install it? [y/N]\n", ver)
			if getConfirmation() {
				if ver, err = z.Install(ver, false, false, true); err != nil {
					return ver, err
				}
			} else {
				return ver, fmt.Errorf("version %s is not installed", ver)
			}
		}
	}

	return ver, z.setBin(ver)
}

// setBin updates the symbolic link 'bin' in the ZVM base directory to point to the specified version's bin directory.
func (z *ZVM) setBin(ver string) error {
	if z.usesXDGSpec() {
		return z.setXDGExecutables(ver)
	}

	// .zvm/master
	versionPath := filepath.Join(z.versionsDir(), ver)
	binDir := z.binDir()

	// Came across https://pkg.go.dev/os#Lstat
	// which is specifically to check symbolic links.
	// Seemed like the more appropriate solution here
	stat, err := os.Lstat(binDir)

	// Actually we need to check if the symbolic link to ~/.zvm/bin
	// exists yet, otherwise we get err:
	//
	// CreateFile C:\Users\gs\.zvm\bin: The system cannot find the file specified.
	//
	// which leads to evaluation of the else case (line 59) and to an early return
	// therefore the the initial symbolic link is never created.
	if stat != nil {
		if err == nil {
			log.Debugf("Removing old %s", binDir)
			if err := os.Remove(binDir); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%w: %s", err, stat.Name())
		}
	}

	if err := meta.Link(versionPath, binDir); err != nil {
		return err
	}

	log.Debug("Use", "version", ver)
	return nil
}

func (z *ZVM) setXDGExecutables(version string) error {
	versionsDir := z.versionsDir()
	for _, name := range []string{"zig", "zls"} {
		source := filepath.Join(versionsDir, version, name)
		destination := filepath.Join(z.binDir(), name)

		_, sourceErr := os.Stat(source)
		if sourceErr != nil && !errors.Is(sourceErr, os.ErrNotExist) {
			return sourceErr
		}

		removed, err := removeZVMManagedLink(destination, versionsDir)
		if err != nil {
			return err
		}

		if errors.Is(sourceErr, os.ErrNotExist) {
			continue
		}

		if !removed {
			if _, err := os.Lstat(destination); err == nil {
				return fmt.Errorf("%s already exists and is not managed by ZVM", destination)
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}

		if err := meta.Link(source, destination); err != nil {
			return err
		}
	}

	log.Debug("Use", "version", version)
	return nil
}

func removeZVMManagedLink(linkPath string, versionsDir string) (bool, error) {
	info, err := os.Lstat(linkPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false, nil
	}

	target, err := os.Readlink(linkPath)
	if err != nil {
		return false, err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(linkPath), target)
	}

	versionsAbs, err := filepath.Abs(versionsDir)
	if err != nil {
		return false, err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false, err
	}
	relative, err := filepath.Rel(versionsAbs, targetAbs)
	if err != nil {
		return false, err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return false, nil
	}

	if err := os.Remove(linkPath); err != nil {
		return false, err
	}
	return true, nil
}

// getConfirmation prompts the user for a yes/no confirmation via stdin.
func getConfirmation() bool {
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	answer := strings.TrimSpace(strings.ToLower(text))
	return answer == "y" || answer == "ye" || answer == "yes"
}

var minZigVersionRe = regexp.MustCompile(`(?m)^\s*\.minimum_zig_version\s*=\s*"([^"]+)"`)

func ExtractMinimumZigVersion() (string, error) {

	// this is the new part.
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "./"
	}
	zigBuildfile := filepath.Join(cwd, "build.zig.zon")
	log.Debug("build.zig.zon check", "cwd", zigBuildfile)

	// only fetches uncommented minimum_zig_versions
	data, err := os.ReadFile(zigBuildfile)
	if err != nil {
		return "", fmt.Errorf("couldn't read build.zig.zon: %w", err)
	}

	matches := minZigVersionRe.FindSubmatch(data)
	if len(matches) < 2 {
		return "", fmt.Errorf("build.zig.zon minimum_zig_version is unparsable. Please provide a valid Zig version as an argument for `use`")
	}
	answer := string(matches[1])
	log.Debug("build.zig.zon exists!", "minimum_zig_version", answer)
	versionArg := strings.TrimPrefix(answer, "v")

	return versionArg, nil
}
