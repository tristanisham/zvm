// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	"golang.org/x/mod/semver"

	"github.com/tristanisham/clr"
)

func (z *ZVM) ListVersions() error {
	if err := z.Clean(); err != nil {
		return err
	}
	cmd := exec.Command("zig", "version")
	var zigVersion strings.Builder
	cmd.Stdout = &zigVersion
	err := cmd.Run()
	if err != nil {
		log.Warn(err)
	}

	version := zigVersion.String()

	installedVersions, err := z.GetInstalledVersions()
	if err != nil {
		return err
	}

	for _, key := range installedVersions {
		if key == strings.TrimSpace(version) || key == "master" && strings.Contains(version, "-dev.") {
			if z.Settings.UseColor {
				// Should just check bin for used version
				fmt.Println(clr.Green(key))
			} else {
				fmt.Printf("%s [x]", key)
			}
		} else {
			fmt.Println(key)
		}
	}

	return nil
}

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

	fmt.Printf("%-12s%-12s%s\n", "Version", "Installed", "ZLS")

	for _, version := range options {
		stripped := version[1:]

		if stripped == "master" {
			continue
		}

		installed := ""
		if slices.Contains(installedVersions, stripped) {
			installed = "[installed]"
		}

		zlsInfo := ""
		if _, ok := zlsVersions[stripped]; ok {
			zlsInfo = "(zls tagged)"
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
			zlsInfo = "(zls tagged)"
		}
		fmt.Printf("%-12s%-12s%s\n", fmt.Sprintf("master (remote) (%s)", remoteVersion), "", zlsInfo)

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
					outDated = "[outdated]"
				}

				fmt.Printf("%-15s (%-15s) %-10s %-10s\n", "master (local)", localVersion, "[installed]", outDated)

			}
		}
	}

	return nil
}
