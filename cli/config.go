package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

func Initialize() *ZVM {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	zvm_path := filepath.Join(home, ".zvm")
	if _, err := os.Stat(zvm_path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Join(zvm_path, "self"), 0775); err != nil {
			log.Fatal(err)
		}
	}

	zvm := &ZVM{
		zvmBaseDir: zvm_path,
	}

	if err := zvm.loadSettings(); err != nil {
		if err.Error() == "settings.json not found" {
			zvm.Settings = Settings{
				UseColor: true,
			}

			out_settings, err := json.MarshalIndent(&zvm.Settings, "", "    ")
			if err != nil {
				log.Warn("Unable to generate settings.json file", err)
			}

			if err := os.WriteFile(filepath.Join(zvm_path, "settings.json"), out_settings, 0755); err != nil {
				log.Warn("Unable to create settings.json file", err)
			}
		}
	}

	zvm.Settings.basePath = filepath.Join(zvm_path, "settings.json")
	return zvm
}

type ZVM struct {
	zvmBaseDir string
	Settings   Settings
}

// A representaiton of the offical json schema for Zig versions
type zigVersionMap = map[string]zigVersion

// LoadMasterVersion takes a zigVersionMap and returns the master disto's version if it's present.
// If it's not, this function returns an empty string.
func LoadMasterVersion(zigMap *zigVersionMap) string {
	if ver, ok := (*zigMap)["master"]["version"].(string); ok {
		return ver
	}
	return ""
}

// A representation of individual Zig versions
type zigVersion = map[string]any

type ZigOnlVersion = map[string][]map[string]string

// func (z *ZVM) loadVersionCache() error {
// 	ver, err := os.ReadFile(filepath.Join(z.zvmBaseDir, "versions.json"))
// 	if err != nil {
// 		return err
// 	}
// 	if err := json.Unmarshal(ver, &z.zigVersions); err != nil {
// 		return err
// 	}
// 	return nil
// }

func (z ZVM) getVersion(version string) error {
	if _, err := os.Stat(filepath.Join(z.zvmBaseDir, version)); err != nil {
		return err
	}
	targetZig := strings.TrimSpace(filepath.Join(z.zvmBaseDir, version, "zig"))
	cmd := exec.Command(targetZig, "version")
	var zigVersion strings.Builder
	cmd.Stdout = &zigVersion
	err := cmd.Run()
	if err != nil {
		log.Warn(err)
	}

	outputVersion := strings.TrimSpace(zigVersion.String())

	log.Debug("getVersion:", "output", outputVersion, "version", version, "program", targetZig)

	if version == outputVersion {
		return nil
	} else {
		if version == "master" {
			return nil
		}
		return fmt.Errorf("version %s is not a released version", version)
	}

}

func (z *ZVM) loadSettings() error {
	set_path := filepath.Join(z.zvmBaseDir, "settings.json")
	if _, err := os.Stat(set_path); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("settings.json not found")
	}

	data, err := os.ReadFile(filepath.Join(z.zvmBaseDir, "settings.json"))
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &z.Settings)
}
