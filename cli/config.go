// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/log"
)

var ErrNoSettings = errors.New("settings.json not found")

// Initialize sets up the ZVM environment, including the base directory
// and settings.json. It creates necessary directories if they don't exist
// and loads the configuration from disk.
func Initialize() *ZVM {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}

	dirs, discoveredSettings, discoverErr := discoverDirectories(home, runtime.GOOS, os.Getenv)
	if discoverErr != nil {
		log.Warn("Unable to read XDG settings; using the default installation", "err", discoverErr)
	}
	if dirs.useXDGSpec {
		warnIfDefaultInstallationExists(filepath.Join(home, ".zvm"))
	}

	if err := ensureDirectories(dirs); err != nil {
		log.Fatal(err)
	}

	zvm := &ZVM{
		baseDir:     dirs.state,
		Directories: dirs,
	}

	if discoveredSettings != nil {
		zvm.Settings = *discoveredSettings
		if err := zvm.Settings.save(); err != nil {
			log.Warn("Unable to update XDG settings.json file", "err", err)
		}
		return zvm
	}

	zvm.Settings.path = dirs.settings

	if err := zvm.loadSettings(); err != nil {
		if errors.Is(err, ErrNoSettings) {
			zvm.Settings = DefaultSettings
			zvm.Settings.path = dirs.settings
			if err := zvm.Settings.save(); err != nil {
				log.Warn("Unable to create settings.json file", "err", err)
			}
		} else {
			log.Warn("Unable to load settings.json file; using defaults", "err", err)
			zvm.Settings = DefaultSettings
			zvm.Settings.path = dirs.settings
		}
	}

	return zvm
}

// ZVM represents the Zig Version Manager and holds its configuration
// and state, including the base directory for installations and settings.
type ZVM struct {
	baseDir     string
	Directories Directories
	Settings    Settings
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

// ZigOnlVersion represents the structure of the Zig version data used by some online tools.
// It maps a version string to a list of platform-specific download info.
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

// validVmuAlis checks if the provided version string is a valid VMU alias.
// Valid aliases are "default" and "mach".
// TODO: Fix typo in function name (Alis -> Alias).
func validVmuAlis(version string) bool {
	return version == "default" || version == "mach"
}

// getVersion determines the actual version string for a given input (e.g., resolving "master").
// It checks if the version is installed and returns an error if it's not a valid release
// or if the installed version doesn't match expectations.
func (z ZVM) getVersion(version string) error {
	root, err := os.OpenRoot(z.versionsDir())
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

// loadSettings loads the ZVM configuration from settings.json.
// It handles missing settings files and ensures empty fields are reset to defaults.
func (z *ZVM) loadSettings() error {
	settings, err := readSettingsFile(z.Settings.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNoSettings
		}
		return err
	}

	z.Settings = settings
	return z.Settings.save()
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
