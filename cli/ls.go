// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"os"
	"os/exec"
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
		switch key.Name() {
		case "settings.json", "bin", "versions.json", "versions-zls.json", "self":
			continue
		default:
			versions = append(versions, key.Name())
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

	options := make([]string, 0, len(zigVersions))

	// add 'v' prefix for sorting.
	for key := range zigVersions {
		options = append(options, "v"+key)
	}

	semver.Sort(options)
	slices.Reverse(options)

	// remove "v" prefix to maintain consistency with zig versioning.
	finalList := options[:0]
	for _, version := range options {
		stripped := version[1:]
		if _, ok := zlsVersions[stripped]; ok {
			stripped += "\t(tagged zls)"
		}
		finalList = append(finalList, stripped)
	}

	fmt.Println(strings.Join(finalList, "\n"))

	return nil
}
