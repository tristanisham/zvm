// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Directories contains the filesystem locations used by ZVM.
type Directories struct {
	data       string
	config     string
	state      string
	cache      string
	bin        string
	self       string
	settings   string
	useXDGSpec bool
}

func defaultDirectories(root string) Directories {
	return Directories{
		data:     root,
		config:   root,
		state:    root,
		cache:    root,
		bin:      filepath.Join(root, "bin"),
		self:     filepath.Join(root, "self"),
		settings: filepath.Join(root, "settings.json"),
	}
}

func xdgDirectories(home string, getenv func(string) string) Directories {
	configHome := absoluteXDGHome(getenv("XDG_CONFIG_HOME"), filepath.Join(home, ".config"))
	dataHome := absoluteXDGHome(getenv("XDG_DATA_HOME"), filepath.Join(home, ".local", "share"))
	stateHome := absoluteXDGHome(getenv("XDG_STATE_HOME"), filepath.Join(home, ".local", "state"))
	cacheHome := absoluteXDGHome(getenv("XDG_CACHE_HOME"), filepath.Join(home, ".cache"))

	return Directories{
		data:       filepath.Join(dataHome, "zvm"),
		config:     filepath.Join(configHome, "zvm"),
		state:      filepath.Join(stateHome, "zvm"),
		cache:      filepath.Join(cacheHome, "zvm"),
		bin:        filepath.Join(home, ".local", "bin"),
		self:       filepath.Join(home, ".local", "bin"),
		settings:   filepath.Join(configHome, "zvm", "settings.json"),
		useXDGSpec: true,
	}
}

func absoluteXDGHome(value string, fallback string) string {
	if value == "" || !filepath.IsAbs(value) {
		return fallback
	}
	return value
}

func discoverDirectories(home string, goos string, getenv func(string) string) (Directories, *Settings, error) {
	if zvmPath := getenv("ZVM_PATH"); zvmPath != "" {
		return defaultDirectories(zvmPath), nil, nil
	}

	defaultPath := filepath.Join(home, ".zvm")
	defaultDirs := defaultDirectories(defaultPath)
	if goos != "linux" {
		return defaultDirs, nil, nil
	}

	xdgDirs := xdgDirectories(home, getenv)
	settings, err := readSettingsFile(xdgDirs.settings)
	if errors.Is(err, os.ErrNotExist) {
		return defaultDirs, nil, nil
	}
	if err != nil {
		return defaultDirs, nil, err
	}
	if !settings.UseXDGSpec {
		return defaultDirs, nil, nil
	}

	return xdgDirs, &settings, nil
}

func ensureDirectories(dirs Directories) error {
	if !dirs.useXDGSpec {
		return os.MkdirAll(dirs.self, 0775)
	}

	for _, dir := range []string{dirs.data, dirs.config, dirs.state, dirs.cache, dirs.bin} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func warnIfDefaultInstallationExists(defaultPath string) {
	if _, err := os.Stat(defaultPath); err != nil {
		return
	}

	fmt.Fprint(os.Stderr, defaultInstallationWarning(defaultPath))
}

func defaultInstallationWarning(defaultPath string) string {
	return fmt.Sprintf(
		"Warning: An existing default ZVM installation was found at %s.\n"+
			"ZVM is using the XDG installation. Remove %s if you no longer need the default installation.\n"+
			"To switch back, remove the XDG installation and rerun install.sh without --use-xdg-spec.\n",
		defaultPath,
		defaultPath,
	)
}

func (z ZVM) versionsDir() string {
	if z.Directories.state != "" {
		return z.Directories.state
	}
	return z.baseDir
}

func (z ZVM) cacheDir() string {
	if z.Directories.cache != "" {
		return z.Directories.cache
	}
	return z.baseDir
}

func (z ZVM) binDir() string {
	if z.Directories.bin != "" {
		return z.Directories.bin
	}
	return filepath.Join(z.baseDir, "bin")
}

func (z ZVM) selfDir() string {
	if z.Directories.self != "" {
		return z.Directories.self
	}
	return filepath.Join(z.baseDir, "self")
}

func (z ZVM) usesXDGSpec() bool {
	return z.Directories.useXDGSpec
}

func readSettingsFile(path string) (Settings, error) {
	settings := DefaultSettings
	settings.path = path

	data, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, err
	}

	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}

	return settings, nil
}
