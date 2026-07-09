// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/log"
	"golang.org/x/mod/semver"

	"github.com/tristanisham/clr"
	"github.com/tristanisham/zvm/cli/meta"
)

// ListVersions prints the installed Zig versions and marks the current version.
func (z *ZVM) ListVersions() error {
	if err := z.Clean(); err != nil {
		return err
	}
	cmd := exec.Command("zig", "version")
	var zigVersion strings.Builder
	cmd.Stdout = &zigVersion
	err := cmd.Run()
	if err != nil {
		log.Debug(err)
	}

	version := zigVersion.String()

	installedVersions, err := z.GetInstalledVersions()
	if err != nil {
		return err
	}

	if len(installedVersions) == 0 {
		cmdHelp := "zvm ls --all"
		if z.Settings.UseColor {
			cmdHelp = clr.Blue(cmdHelp)
		}
		fmt.Printf("No local Zig installs. Run `%s` to list all available-to-install versions of Zig.\n", cmdHelp)
	}

	var aliasesByVersion map[string][]string
	if z.db == nil {
		log.Debug("alias database unavailable; listing versions without aliases")
	} else {
		aliasesByVersion, err = z.aliasesByVersion(context.Background())
		if err != nil {
			return err
		}
	}

	for _, key := range installedVersions {
		active := key == strings.TrimSpace(version) || key == "master" && strings.Contains(version, "-dev.")
		fmt.Println(formatVersionLine(key, formatAliasColumn(aliasesByVersion[key]), active, z.Settings.UseColor))
	}

	return nil
}

// formatVersionLine renders a single line of `zvm ls` output for an installed
// version, optionally marking it active and appending an alias column.
func formatVersionLine(version, aliasCol string, active, useColor bool) string {
	var line string
	switch {
	case active && useColor:
		line = clr.Green(version)
	case active && !useColor:
		line = version + " [x]"
	default:
		line = version
	}

	if aliasCol != "" {
		if useColor {
			aliasCol = clr.Blue(aliasCol)
		}
		line += "  " + aliasCol
	}

	return line
}

// GetInstalledVersions returns a slice of strings containing the names of
// all installed Zig versions found in the base directory.
func (z *ZVM) GetInstalledVersions() ([]string, error) {
	dir, err := os.ReadDir(z.baseDir)
	if err != nil {
		return nil, err
	}

	versions := make([]string, 0, len(dir))
	for _, key := range dir {
		if key.IsDir() {
			switch key.Name() {
			case "bin", "self":
				continue
			default:
				versions = append(versions, key.Name())
			}
		}
	}
	return versions, nil
}

type RemoteVersionJSON struct {
	Version       string   `json:"version"`
	Installed     bool     `json:"installed"`
	ZLS           bool     `json:"zls"`
	Aliases       []string `json:"aliases,omitempty"`
	RemoteVersion string   `json:"remoteVersion,omitempty"`
	LocalVersion  string   `json:"localVersion,omitempty"`
	Outdated      bool     `json:"outdated,omitempty"`
}

// ListRemoteAvailable lists all available Zig versions from the remote version map,
// indicating which ones are already installed and which have ZLS support.
func (z ZVM) ListRemoteAvailable() error {
	zigVersions, err := z.fetchVersionMap()
	if err != nil {
		return err
	}

	zlsVersions, err := z.fetchZlsTaggedVersionMap()
	if err != nil {
		return err
	}

	installedVersions, err := z.GetInstalledVersions()
	if err != nil {
		return err
	}

	options := make([]string, 0, len(zigVersions))

	// add 'v' prefix for sorting.
	for key := range zigVersions {
		options = append(options, "v"+key)
	}

	semver.Sort(options)
	slices.Reverse(options)

	fmt.Printf("%s%s%s\n", meta.TableVersionHeaderStyle.Render("Version"), meta.TableInstalledHeaderStyle.Render("Installed"), meta.TableHeaderStyle.Render("ZLS"))

	for _, version := range options {
		stripped := version[1:]

		if stripped == "master" {
			continue
		}

		installed := ""
		if slices.Contains(installedVersions, stripped) {
			if z.Settings.UseColor {
				installed = "installed"
				coloredStr := meta.TableInstalledVersionStyle.Render(installed)
				// keep the extra space. It fixes a rendering bug.
				installed = fmt.Sprintf("[%s] ", coloredStr)
			} else {
				installed = "[installed]"

			}
		}

		zlsInfo := ""
		if _, ok := zlsVersions[stripped]; ok {
			zlsInfo = "(tagged)"
		}

		fmt.Printf("%-12s%-12s%s\n", stripped, installed, zlsInfo)
	}

	if _, ok := zigVersions["master"]; ok {
		var remoteVersion string
		if master, ok := zigVersions["master"]; ok {
			if versionInfo, ok := master["version"].(string); ok {
				remoteVersion = versionInfo
			}
		}

		zlsInfo := ""
		if _, ok := zlsVersions["master"]; ok {
			zlsInfo = "(tagged)"
		}
		remoteLabel := "remote"
		if z.Settings.UseColor {
			remoteLabel = meta.TableRemoteVersionStyle.Render(remoteLabel)
		}
		fmt.Printf("%-12s%-12s%s\n", fmt.Sprintf("master (%s) (%s)", remoteLabel, remoteVersion), "", zlsInfo)

		// Check if master is installed and print local version
		if slices.Contains(installedVersions, "master") {
			targetZig := strings.TrimSpace(filepath.Join(z.baseDir, "master", "zig"))
			cmd := exec.Command(targetZig, "version")
			var zigVersion strings.Builder
			cmd.Stdout = &zigVersion
			err := cmd.Run()
			if err != nil {
				log.Warn(err)
			} else {
				localVersion := strings.TrimSpace(zigVersion.String())

				outDated := ""
				if localVersion != remoteVersion {
					if z.Settings.UseColor {
						outDated = fmt.Sprintf("[%s] ", meta.TableOutdatedVersionStyle.Render("outdated"))
					} else {
						outDated = "[outdated]"
					}
				}

				localLabel := "local"
				installed := "[installed]"
				if z.Settings.UseColor {
					localLabel = meta.TableLocalVersionStyle.Render(localLabel)
					installed = fmt.Sprintf("[%s] ", meta.TableInstalledVersionStyle.Render("installed"))
				}

				fmt.Println("--------------------------------------------")
				fmt.Printf("%-15s (%-15s) %-10s %-10s\n", fmt.Sprintf("master (%s)", localLabel), localVersion, installed, outDated)

			}
		}
	}

	return nil
}

func (z ZVM) ListRemoteAvailableJSON() error {
	zigVersions, err := z.fetchVersionMap()
	if err != nil {
		return err
	}

	zlsVersions, err := z.fetchZlsTaggedVersionMap()
	if err != nil {
		return err
	}

	installedVersions, err := z.GetInstalledVersions()
	if err != nil {
		return err
	}

	aliasesByVersion, err := z.aliasesByVersion(context.Background())
	if err != nil {
		return err
	}

	options := make([]string, 0, len(zigVersions))
	for key := range zigVersions {
		options = append(options, "v"+key)
	}

	semver.Sort(options)
	slices.Reverse(options)

	versions := make([]RemoteVersionJSON, 0, len(options))
	for _, version := range options {
		stripped := version[1:]
		if stripped == "master" {
			continue
		}

		_, hasZLS := zlsVersions[stripped]
		versions = append(versions, RemoteVersionJSON{
			Version:   stripped,
			Installed: slices.Contains(installedVersions, stripped),
			ZLS:       hasZLS,
			Aliases:   aliasesByVersion[stripped],
		})
	}

	if master, ok := zigVersions["master"]; ok {
		var remoteVersion string
		if versionInfo, ok := master["version"].(string); ok {
			remoteVersion = versionInfo
		}

		_, hasZLS := zlsVersions["master"]
		masterVersion := RemoteVersionJSON{
			Version:       "master",
			Installed:     slices.Contains(installedVersions, "master"),
			ZLS:           hasZLS,
			Aliases:       aliasesByVersion["master"],
			RemoteVersion: remoteVersion,
		}

		if masterVersion.Installed {
			targetZig := strings.TrimSpace(filepath.Join(z.baseDir, "master", "zig"))
			cmd := exec.Command(targetZig, "version")
			var zigVersion strings.Builder
			cmd.Stdout = &zigVersion
			if err := cmd.Run(); err != nil {
				log.Warn(err)
			} else {
				localVersion := strings.TrimSpace(zigVersion.String())
				masterVersion.LocalVersion = localVersion
				masterVersion.Outdated = localVersion != remoteVersion
			}
		}

		versions = append(versions, masterVersion)
	}

	return json.NewEncoder(os.Stdout).Encode(versions)
}

func (z *ZVM) aliasesByVersion(ctx context.Context) (map[string][]string, error) {
	aliases, err := z.ListAliases(ctx)
	if err != nil {
		return nil, err
	}

	aliasesByVersion := make(map[string][]string, len(aliases))
	for _, alias := range aliases {
		aliasesByVersion[alias.Value] = append(aliasesByVersion[alias.Value], alias.Key)
	}

	for version := range aliasesByVersion {
		sort.Strings(aliasesByVersion[version])
	}

	return aliasesByVersion, nil
}

func formatAliasColumn(aliases []string) string {
	const maxAliases = 3

	if len(aliases) == 0 {
		return ""
	}

	if len(aliases) <= maxAliases {
		return strings.Join(aliases, ", ")
	}

	return fmt.Sprintf("%s... (%d)", strings.Join(aliases[:maxAliases], ", "), len(aliases))
}
