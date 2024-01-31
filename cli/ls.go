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
	dir, err := os.ReadDir(z.baseDir)
	if err != nil {
		return err
	}

	for _, key := range dir {
		switch key.Name() {
		case "settings.json", "bin", "versions.json", "self":
			continue
		default:
			if key.Name() == strings.TrimSpace(version) {
				if z.Settings.UseColor {
					// Should just check bin for used version
					fmt.Println(clr.Green(key.Name()))
				} else {
					fmt.Printf("%s [x]", key.Name())
				}
			} else {
				fmt.Println(key.Name())
			}
		}
	}

	return nil
}

func (z ZVM) ListRemoteAvailable() error {
	versions, err := z.fetchVersionMap()
	if err != nil {
		return err
	}

	options := make([]string, 0)

	for key := range versions {
		if key == "master" {
			// Removes master for sorting. Must add back in later.
			continue
		}

		options = append(options, "v"+key)
	}

	semver.Sort(options)
	slices.Reverse(options)

	// Remove "v" prefix to maintain consistency with zig versioning
	newOptions := options[:0]
	for _, version := range options {
		newOptions = append(newOptions, version[1:])
	}

	finalList := []string{"master"}
	finalList = append(finalList, newOptions...)

	fmt.Println(strings.Join(finalList, "\n"))

	return nil
}
