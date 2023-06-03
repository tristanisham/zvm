package cli

import (
	"fmt"
	"github.com/charmbracelet/log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tristanisham/clr"
)

func (z *ZVM) ListVersions() error {
	iv, err := z.installVersions()
	if err != nil {
		return err
	}

	cmd := exec.Command("zig", "version")
	var zigVersion strings.Builder
	cmd.Stdout = &zigVersion
	err = cmd.Run()
	if err != nil {
		log.Warn(err)
	}

	var version string = zigVersion.String()
	if strings.Contains(version, "-dev") {
		version = "master"
	}

	for key := range *iv {
		if key == strings.TrimSpace(version) {
			if z.Settings.UseColor {
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

func (z ZVM) installVersions() (*map[string]string, error) {
	dir, err := os.ReadDir(z.zvmBaseDir)
	if err != nil {
		return nil, err
	}

	if err := z.loadVersionCache(); err != nil {
		return nil, err
	}

	result := make(map[string]string)

	for _, entry := range dir {
		if !entry.IsDir() {
			continue
		}

		if _, ok := z.zigVersions[entry.Name()]; ok {
			result[entry.Name()] = filepath.Join(z.zvmBaseDir, entry.Name())
		}
	}

	return &result, nil
}
