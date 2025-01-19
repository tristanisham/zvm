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

	zvm := &ZVM{
		dataDir:   ZvmDataDirPath(home),
		stateDir:  ZvmStateDirPath(home),
		configDir: ZvmConfigDirPath(home),
		binDir:    ZvmBinDirPath(home),
		cacheDir:  ZvmCacheDirPath(home),
	}

	// Loop through the zvm fields and make the directories if they don't exist
	for _, dir := range []string{zvm.dataDir, zvm.stateDir, zvm.configDir, zvm.binDir, zvm.cacheDir} {
		if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
			if err := os.MkdirAll(dir, 0775); err != nil {
				log.Fatal(err)
			}
		}
	}

	log.Debug("Initialize:", "dataDir", zvm.dataDir)
	log.Debug("Initialize:", "stateDir", zvm.stateDir)
	log.Debug("Initialize:", "configDir", zvm.configDir)
	log.Debug("Initialize:", "binDir", zvm.binDir)
	log.Debug("Initialize:", "cacheDir", zvm.cacheDir)

	if _, err := os.Stat(zvm.dataDir); errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(filepath.Join(zvm.dataDir, "self"), 0775); err != nil {
			log.Fatal(err)
		}
	}

	zvm.Settings.path = filepath.Join(zvm.configDir, "settings.json")
	log.Debug("Initialize:", "Settings path", zvm.Settings.path)

	if err := zvm.loadSettings(); err != nil {
		if errors.Is(err, ErrNoSettings) {
			// We need to get a copy of this path, then restore it after writing
			// default settings out
			settings_path := zvm.Settings.path
			zvm.Settings = Settings{
				UseColor:           true,
				VersionMapUrl:      "https://ziglang.org/download/index.json",
				AlwaysForceInstall: false,
				ZlsVMU:             "https://releases.zigtools.org/",
			}

			out_settings, err := json.MarshalIndent(&zvm.Settings, "", "    ")
			if err != nil {
				log.Warn("Unable to generate settings.json file", err)
			}

			if err := os.WriteFile(settings_path, out_settings, 0755); err != nil {
				log.Warn("Unable to create settings.json file", err)
			}
			zvm.Settings.path = settings_path
		}
	}

	return zvm
}

type ZVM struct {
	// We place ourselves in the data directory
	// The installer should make a symlink from the bin directory to the data
	// directory
	dataDir string
	// This is where settings.json lives
	configDir string
	// We place zig versions under the state directory
	stateDir string
	// We place current zig/zls in here
	binDir string
	// Used for storage of temporary downloads, etc
	cacheDir string
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

func (z ZVM) zigPath() (string, error) {
	zig := filepath.Join(z.binDir, "zig")
	log.Debug("zigPath", "zig", zig)
	if _, err := os.Stat(zig); err != nil {
		return "", err
	}

	return zig, nil
}

func (z ZVM) getVersion(version string) error {
	if _, err := os.Stat(filepath.Join(z.stateDir, version)); err != nil {
		return err
	}

	targetZig := strings.TrimSpace(filepath.Join(z.stateDir, version, "zig"))
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
		if _, statErr := os.Stat(targetZig); statErr == nil || version == "master" {
			return nil
		}
		return fmt.Errorf("version %s is not a released version", version)
	}
}

func (z *ZVM) loadSettings() error {
	set_path := z.Settings.path
	if _, err := os.Stat(set_path); errors.Is(err, os.ErrNotExist) {
		return ErrNoSettings
	}

	data, err := os.ReadFile(set_path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &z.Settings)
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
