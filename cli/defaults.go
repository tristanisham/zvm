package cli

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/log"
)

const compiledOs = runtime.GOOS

func zvmDirectories(home string, useRuntime bool) Directories {
	var os string
	if useRuntime {
		os = runtime.GOOS
	} else {
		os = compiledOs
	}
	switch os {
	case "windows":
		return windowsDirectories(home)
	case "darwin":
		return darwinDirectories(home)
	case "aix", "dragonfly", "freebsd", "linux", "nacl", "netbsd", "openbsd", "solaris":
		return unixDirectories(home)
	case "plan9":
		return plan9Directories(home)
	default:
		// we are on a weird OS. Back up to what we were compiled with
		if useRuntime {
			log.Infof("Unknown runtime operating system %s. Will determine paths based on compiled os %s", os, compiledOs)
			// Call ourselves based on compiled OS
			return zvmDirectories(home, false)
		} else {
			// Well now we have no idea...home?
			log.Warnf("Unknown operating system %s. Assuming Unix derivative for install paths", os)
			return unixDirectories(home)
		}
	}
}

type PathType int

const (
	NativePathing   PathType = 0
	ZvmPath         PathType = 1
	ExistingInstall PathType = 2
)

func zvmPathDirectories(home string) (Directories, PathType) {
	zvmInstallPath := os.Getenv("ZVM_INSTALL")
	if zvmPath := os.Getenv("ZVM_PATH"); zvmPath != "" {
		if zvmInstallPath == "" {
			zvmInstallPath = filepath.Join(zvmPath, "self")
		}
		return Directories{
			data:   zvmPath,
			config: zvmPath,
			state:  zvmPath,
			cache:  zvmPath,
			bin:    zvmPath,
			self:   zvmInstallPath,
		}, ZvmPath
	}
	// Look for an existing installation and return that...
	existingPath := filepath.Join(home, ".zvm")
	if info, err := os.Stat(existingPath); err == nil && info.IsDir() {
		log.Debugf("Using existing zvm installation in %s", existingPath)
		if zvmInstallPath == "" {
			zvmInstallPath = filepath.Join(existingPath, "self")
		}
		return Directories{
			data:   existingPath,
			config: existingPath,
			state:  existingPath,
			cache:  existingPath,
			bin:    filepath.Join(existingPath, "bin"),
			self:   zvmInstallPath,
		}, ExistingInstall
	}
	return Directories{}, NativePathing
}
func darwinDirectories(home string) Directories {
	rc, pathType := Directories{}, NativePathing
	if rc, pathType = zvmPathDirectories(home); pathType == NativePathing {
		rc.data = filepath.Join(home, "Library", "Application Support", "zvm")
		rc.config = filepath.Join(home, "Library", "Preferences", "zvm")
		rc.state = rc.data
		rc.cache = filepath.Join(home, "Library", "Caches", "zvm")
	}
	if pathType != ExistingInstall {
		rc.bin = filepath.Join(home, ".local", "bin")
		rc.self = rc.bin
	}
	return rc
}

func windowsDirectories(home string) Directories {
	if rc, pathType := zvmPathDirectories(home); pathType != NativePathing {
		return rc
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(home, "AppData", "Roaming")
	}

	return Directories{
		data:   filepath.Join(localAppData, "zvm"),
		config: filepath.Join(appData, "zvm"),
		state:  filepath.Join(localAppData, "zvm"),
		cache:  filepath.Join(localAppData, "zvm", "cache"),
		bin:    filepath.Join(localAppData, "bin"),
		self:   filepath.Join(localAppData, "bin"),
	}
}

func plan9Directories(home string) Directories {
	rc, pathType := Directories{}, NativePathing
	if rc, pathType = zvmPathDirectories(home); pathType == NativePathing {
		rc.data = filepath.Join(home, ".zvm")
		rc.config = rc.data
		rc.state = rc.data
		rc.cache = filepath.Join(rc.data, "cache")
	}
	if pathType != ExistingInstall {
		rc.bin = filepath.Join(home, "bin")
		rc.self = rc.bin
	}
	return rc
}

func unixDirectories(home string) Directories {
	rc, pathType := Directories{}, NativePathing
	if rc, pathType = zvmPathDirectories(home); pathType == NativePathing {
		if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
			rc.data = filepath.Join(xdgDataHome, "zvm")
		} else {
			rc.data = filepath.Join(home, ".local", "share", "zvm")
		}

		if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
			rc.config = filepath.Join(xdgConfigHome, "zvm")
		} else {
			rc.config = filepath.Join(home, ".config", "zvm")
		}

		if xdgStateHome := os.Getenv("XDG_STATE_HOME"); xdgStateHome != "" {
			rc.state = filepath.Join(xdgStateHome, "zvm")
		} else {
			rc.state = filepath.Join(home, ".local", "state", "zvm")
		}

		if xdgCacheHome := os.Getenv("XDG_CACHE_HOME"); xdgCacheHome != "" {
			rc.cache = filepath.Join(xdgCacheHome, "zvm")
		} else {
			rc.cache = filepath.Join(home, ".cache", "zvm")
		}
	}

	if pathType != ExistingInstall {
		if xdgBinHome := os.Getenv("XDG_BIN_HOME"); xdgBinHome != "" {
			rc.bin = xdgBinHome
		} else {
			rc.bin = filepath.Join(home, ".local", "bin")
		}
		rc.self = rc.bin
	}
	return rc
}
