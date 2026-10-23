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

	"strings"

	"github.com/charmbracelet/log"
	"github.com/tristanisham/zvm/cli/meta"
)

// Use switches the active Zig version to the specified one.
// If the version is not installed, it prompts the user to install it.
// Returns the resolved version string (which may differ from input if shorthand was used).
//
// If the current shell has a per-shell link (see `zvm env`), only that link is
// repointed; otherwise the global ~/.zvm/bin link is.
func (z *ZVM) Use(ver string) (string, error) {
	resolved, _, err := z.UseVersion(ver, false)
	return resolved, err
}

// UseVersion is Use with control over scope. With global set, or when the
// current shell has no per-shell link, it repoints the global ~/.zvm/bin link
// exactly as Use always has. Otherwise it repoints only the link named by
// ZVM_MULTISHELL_PATH and reports perShell as true.
func (z *ZVM) UseVersion(ver string, global bool) (resolved string, perShell bool, err error) {
	if err := validateVersionTarget(ver, "use"); err != nil {
		return ver, false, err
	}

	shellLink := ""
	if !global {
		link, ok, linkErr := ActiveMultishellLink()
		switch {
		case linkErr != nil:
			log.Warn("ignoring "+MultishellEnv+"; changing the global version instead", "err", linkErr)
		case ok:
			shellLink = link
		}
	}

	// Resolve shorthand against locally installed versions
	if installedVersions, err := z.GetInstalledVersions(); err == nil {
		if resolved, resolveErr := resolveVersionShorthand(ver, installedVersions); resolveErr == nil && resolved != ver {
			log.Debug("resolved version shorthand", "input", ver, "resolved", resolved)
			fmt.Printf("Resolved %q to %s\n", ver, resolved)
			ver = resolved
		}
	}
	if err := checkReservedVersion(ver, "use"); err != nil {
		return ver, false, err
	}

	if err := z.getVersion(ver); err != nil {
		if errors.Is(err, os.ErrNotExist) {

			fmt.Printf("It looks like %s isn't installed. Would you like to install it? [y/N]\n", ver)
			if getConfirmation() {
				// Install must not switch versions itself: UseVersion repoints
				// the right link below, and a second switch would print a second,
				// conflicting message.
				if ver, err = z.Install(ver, false, false, true, true); err != nil {
					return ver, false, err
				}
			} else {
				return ver, false, fmt.Errorf("version %s is not installed", ver)
			}
		} else {
			return ver, false, err
		}
	}

	if shellLink != "" {
		return ver, true, z.setShellLink(shellLink, ver)
	}

	return ver, false, z.setBin(ver)
}

// setBin updates the symbolic link 'bin' in the ZVM base directory to point to the specified version's bin directory.
func (z *ZVM) setBin(ver string) error {
	// .zvm/master
	version_path := filepath.Join(z.baseDir, ver)
	binDir := filepath.Join(z.baseDir, "bin")

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

	if err := meta.Link(version_path, binDir); err != nil {
		return err
	}

	log.Debug("Use", "version", ver)
	return nil
}

// getConfirmation prompts the user for a yes/no confirmation via stdin.
func getConfirmation() bool {
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	answer := strings.TrimSpace(strings.ToLower(text))
	return answer == "y" || answer == "ye" || answer == "yes"
}

// ExtractMinimumZigVersion reads `.minimum_zig_version` from build.zig.zon in
// the current directory only. It deliberately does not search parent
// directories; see FindVersionSource for the recursive lookup used by
// `zvm env --use-on-cd`.
func ExtractMinimumZigVersion() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "./"
	}
	return ExtractMinimumZigVersionIn(cwd)
}

// ExtractMinimumZigVersionIn reads `.minimum_zig_version` from dir/build.zig.zon.
func ExtractMinimumZigVersionIn(dir string) (string, error) {
	zigBuildfile := filepath.Join(dir, "build.zig.zon")
	log.Debug("build.zig.zon check", "cwd", zigBuildfile)

	// only fetches uncommented minimum_zig_versions
	data, err := os.ReadFile(zigBuildfile)
	if err != nil {
		return "", fmt.Errorf("couldn't read build.zig.zon: %w", err)
	}

	versionArg, ok := parseMinimumZigVersion(data)
	if !ok {
		return "", fmt.Errorf("build.zig.zon minimum_zig_version is unparsable. Please provide a valid Zig version as an argument for `use`")
	}
	log.Debug("build.zig.zon exists!", "minimum_zig_version", versionArg)

	return versionArg, nil
}
