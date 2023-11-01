package cli

import (
	"fmt"
	"os"
	"os/exec"
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
	dir, err := os.ReadDir(z.zvmBaseDir)
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
	versions, err := z.fetchOfficialVersionMap()
	if err != nil {
		return err
	}

	options := make([]string, 0)

	for key := range versions {
		if key == "master" {
			// Removes master for sorting. Must add back in later.
			continue
		}

		options = append(options, key)
	}

	semver.Sort(options)

	finalList := []string{"master"}
	finalList = append(finalList, options...)

	fmt.Println(strings.Join(finalList, "\n"))

	return nil
}