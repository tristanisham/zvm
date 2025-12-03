// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

var ErrNoSettings = errors.New("settings.json not found")

func Initialize() *ZVM {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	zvmPath := os.Getenv("ZVM_PATH")
	if zvmPath == "" {
		zvmPath = filepath.Join(home, ".zvm")
	}

	if _, err := os.Stat(zvmPath); errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(filepath.Join(zvmPath, "self"), 0775); err != nil {
			log.Fatal(err)
		}
	}

	zvm := &ZVM{
		baseDir: zvmPath,
	}

	zvm.Settings.path = filepath.Join(zvmPath, "settings.json")

	if err := zvm.loadSettings(); err != nil {
		if errors.Is(err, ErrNoSettings) {
			zvm.Settings = DefaultSettings

			outSettings, err := json.MarshalIndent(&zvm.Settings, "", "    ")
			if err != nil {
				log.Warn("Unable to generate settings.json file", err)
			}

			if err := os.WriteFile(filepath.Join(zvmPath, "settings.json"), outSettings, 0755); err != nil {
				log.Warn("Unable to create settings.json file", err)
			}
		}
	}

	return zvm
}

type ZVM struct {
	baseDir  string
	Settings Settings
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

//	func (z *ZVM) loadVersionCache() error {
//		ver, err := os.ReadFile(filepath.Join(z.zvmBaseDir, "versions.json"))
//		if err != nil {
//			return err
//		}
//		if err := json.Unmarshal(ver, &z.zigVersions); err != nil {
//			return err
//		}
//		return nil
//	}
//
// TODO switch to error so we can handle common typos. Make it return an (error, bool)
func validVmuAlis(version string) bool {
	return version == "default" || version == "mach"
}

func (z ZVM) getVersion(version string) error {
	
	root, err := os.OpenRoot(z.baseDir)
	if err != nil {
		return err
	}

	defer root.Close()

	if _, err := root.Stat(version); err != nil {
		return err
	}

	

	targetZig := strings.TrimSpace(filepath.Join(root.Name(), version, "zig"))
	cmd := exec.Command(targetZig, "version")
	var zigVersion strings.Builder
	cmd.Stdout = &zigVersion
	err = cmd.Run()
	if err != nil {
		log.Warn(err)
	}

	outputVersion := strings.TrimSpace(zigVersion.String())

	log.Debug("getVersion:", "output", outputVersion, "version", version, "program", targetZig)

	if version == outputVersion {
		return nil
	} else {
		if _, statErr := root.Stat(targetZig); statErr == nil || version == "master" {
			return nil
		}
		return fmt.Errorf("version %s is not a released version", version)
	}
}

func (z *ZVM) loadSettings() error {
	setPath := z.Settings.path
	if _, err := os.Stat(setPath); errors.Is(err, os.ErrNotExist) {
		return ErrNoSettings
	}

	data, err := os.ReadFile(setPath)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(data, &z.Settings); err != nil {
		return err
	}

	return z.Settings.ResetEmpty()
}

// func (z *ZVM) AlertIfUpgradable() {
// 	if !z.Settings.StartupCheckUpgrade {
// 		return
// 	}
// 	log.Debug("Checking for upgrade on startup is enabled")
// 	upgradable, tagName, err := CanIUpgrade()
// 	if err != nil {
// 		log.Info("failed new zvm version check")
// 	}

// 	if upgradable {
// 		coloredText := "zvm upgrade"
// 		if z.Settings.UseColor {
// 			coloredText = clr.Blue("zvm upgrade")
// 		}

// 		fmt.Printf("There's a new version of ZVM (%s).\n Run '%s' to install it!\n", tagName, coloredText)
// 	}
// }

// func (z *ZVM) ConflictCheck(file string) (string, error) {
// 	zls, err := exec.LookPath("zls")
// 	if err != nil {
// 		return "", err
// 	}

// 	linuxPath := filepath.Join(z.baseDir,"bin/zls")
// 	if _, err := os.Stat(linuxPath); err == nil {

// 	}
// 	return zls, nil
// }
