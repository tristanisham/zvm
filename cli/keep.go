// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

// keptMaster describes an installed master build that should survive the
// install replacing it.
type keptMaster struct {
	alias   string
	version string // directory the old build moves to, named after its version
	exists  bool   // that directory is already installed, so nothing moves
}

// planKeepMaster decides, before anything is downloaded, how the installed
// master build will be kept under the name alias. It returns nil when there is
// nothing to keep: master is not installed, or it already is remoteVersion.
func (z *ZVM) planKeepMaster(alias, remoteVersion string) (*keptMaster, error) {
	// An alias shadows the version it names (see resolveVersionArg), so
	// "master" would make `zvm use master` select the old build.
	if alias == "master" || isReservedName(alias) {
		return nil, fmt.Errorf("%w: %q cannot be used as a name for the old master build", ErrInvalidInput, alias)
	}
	// Fail on an unreadable alias file now, not after the download.
	if _, err := z.readAliases(); err != nil {
		return nil, err
	}

	zig, err := z.zigExecutable("master")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("No master build is installed; nothing to keep.")
			return nil, nil
		}
		return nil, err
	}

	out, err := exec.Command(zig, "version").Output()
	if err != nil {
		return nil, fmt.Errorf("could not read the version of the installed master build: %w", err)
	}
	oldVersion := strings.TrimSpace(string(out))
	if err := validateVersionTarget(oldVersion, "keep master"); err != nil {
		return nil, fmt.Errorf("installed master reports an unusable version %q: %w", oldVersion, err)
	}
	if oldVersion == "master" || isReservedName(oldVersion) {
		return nil, fmt.Errorf("%w: installed master reports version %q", ErrInvalidInput, oldVersion)
	}

	if oldVersion == remoteVersion {
		fmt.Printf("Installed master is already %s; nothing to keep.\n", oldVersion)
		return nil, nil
	}

	plan := &keptMaster{alias: alias, version: oldVersion}
	if _, err := os.Lstat(filepath.Join(z.baseDir, oldVersion)); err == nil {
		plan.exists = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return plan, nil
}

// keepMaster moves the installed master build out of the way of the new one
// and saves its alias. The caller must hold the store lock.
func (z *ZVM) keepMaster(k *keptMaster) error {
	if !k.exists {
		if err := os.Rename(filepath.Join(z.baseDir, "master"), filepath.Join(z.baseDir, k.version)); err != nil {
			return fmt.Errorf("could not keep the old master build as %s: %w", k.version, err)
		}
	}

	aliases, err := z.readAliases()
	if err == nil {
		aliases[k.alias] = k.version
		err = z.writeAliases(aliases)
	}
	if err != nil {
		// The old build is already safe; only its name is missing.
		log.Warn("kept the old master build but could not save its alias", "version", k.version, "err", err)
		fmt.Printf("Run `zvm alias %s %s` to name it.\n", k.alias, k.version)
		return nil
	}

	fmt.Printf("Kept the old master build as %s (alias %q)\n", k.version, k.alias)
	return nil
}
