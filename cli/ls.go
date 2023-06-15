package cli

import (
	"fmt"
	"github.com/charmbracelet/log"
	"os"
	"os/exec"
	"strings"

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
		case "settings.json", "bin", "versions.json":
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

// func (z ZVM) installVersions() (*map[string]string, error) {
// 	dir, err := os.ReadDir(z.zvmBaseDir)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if err := z.loadVersionCache(); err != nil {
// 		return nil, err
// 	}

// 	result := make(map[string]string)

// 	for _, entry := range dir {
// 		if !entry.IsDir() {
// 			continue
// 		}

// 		if _, ok := z.zigVersions[entry.Name()]; ok {
// 			result[entry.Name()] = filepath.Join(z.zvmBaseDir, entry.Name())
// 		}
// 	}

// 	return &result, nil
// }
