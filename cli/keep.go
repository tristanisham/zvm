// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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
	// replaces is the value the user agreed to overwrite when alias already
	// existed; empty when the alias is new.
	replaces string
}

// confirmAliasUpdate asks whether an existing alias may be repointed. Tests
// replace it to answer without a terminal.
var confirmAliasUpdate = func(alias, current, version string) bool {
	fmt.Printf("Alias %q already points to %s. Update it to the old master build %s? [Y/n]\n", alias, current, version)
	return readConfirmDefaultYes(os.Stdin)
}

// readConfirmDefaultYes reads a yes/no answer where an empty line means yes.
// Closed input means no, so a script without a terminal never overwrites.
func readConfirmDefaultYes(r io.Reader) bool {
	text, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && text == "" {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(text)) {
	case "", "y", "ye", "yes":
		return true
	}
	return false
}

// planKeepMaster decides, before anything is downloaded, how the installed
// master build will be kept under the name alias. It returns nil when there is
// nothing to keep: master is not installed, or it already is remoteVersion.
func (z *ZVM) planKeepMaster(alias, remoteVersion string) (*keptMaster, error) {
	// An alias shadows the version it names (see resolveVersionArg), so
	// "master" would make `zvm use master` select the old build.
	if err := validateVersionTarget(alias, "keep master alias"); err != nil {
		return nil, err
	}
	if strings.EqualFold(alias, "master") || isReservedName(alias) {
		return nil, fmt.Errorf("%w: %q cannot be used as a name for the old master build", ErrInvalidInput, alias)
	}
	// Fail on an unreadable alias file now, not after the download.
	aliases, err := z.readAliases()
	if err != nil {
		return nil, err
	}

	oldVersion, err := z.installedMasterVersion()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("No master build is installed; nothing to keep.")
			return nil, nil
		}
		return nil, err
	}

	if oldVersion == remoteVersion {
		fmt.Printf("Installed master is already %s; nothing to keep.\n", oldVersion)
		return nil, nil
	}

	if _, err := os.Lstat(filepath.Join(z.baseDir, oldVersion)); err == nil {
		return nil, fmt.Errorf("cannot keep master as %q: %w", oldVersion, os.ErrExist)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	plan := &keptMaster{alias: alias, version: oldVersion}
	// Ask before downloading, and only when the alias would actually change.
	if current, exists := aliases[alias]; exists {
		if !confirmAliasUpdate(alias, current, oldVersion) {
			return nil, fmt.Errorf("alias %q already exists: %w", alias, os.ErrExist)
		}
		plan.replaces = current
	}
	return plan, nil
}

// installedMasterVersion runs the installed master zig and returns the version
// it reports, checked to be usable as a directory name.
func (z *ZVM) installedMasterVersion() (string, error) {
	zig, err := z.zigExecutable("master")
	if err != nil {
		return "", err
	}
	out, err := exec.Command(zig, "version").Output()
	if err != nil {
		return "", fmt.Errorf("could not read the version of the installed master build: %w", err)
	}
	version := strings.TrimSpace(string(out))
	if err := validateVersionTarget(version, "keep master"); err != nil {
		return "", fmt.Errorf("installed master reports an unusable version %q: %w", version, err)
	}
	if version == "master" || isReservedName(version) {
		return "", fmt.Errorf("%w: installed master reports version %q", ErrInvalidInput, version)
	}
	return version, nil
}

// keepMaster moves the installed master build out of the way of the new one
// and saves its alias. The caller must hold the store lock.
func (z *ZVM) keepMaster(k *keptMaster) error {
	aliases, err := z.readAliases()
	if err != nil {
		return err
	}
	// The user only agreed to overwrite the value they were shown.
	if current, exists := aliases[k.alias]; exists && (k.replaces == "" || current != k.replaces) {
		return fmt.Errorf("alias %q changed during install: %w", k.alias, os.ErrExist)
	}
	if _, err := os.Lstat(filepath.Join(z.baseDir, k.version)); err == nil {
		return fmt.Errorf("cannot keep master as %q: %w", k.version, os.ErrExist)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// Another zvm may have replaced master since planning; the directory
	// name must match the build that actually moves.
	if current, err := z.installedMasterVersion(); err != nil {
		return err
	} else if current != k.version {
		return fmt.Errorf("installed master changed from %s to %s during install; nothing was moved", k.version, current)
	}
	if err := os.Rename(filepath.Join(z.baseDir, "master"), filepath.Join(z.baseDir, k.version)); err != nil {
		return fmt.Errorf("could not keep the old master build as %s: %w", k.version, err)
	}

	aliases[k.alias] = k.version
	if err := z.writeAliases(aliases); err != nil {
		// The old build is already safe; only its name is missing.
		log.Warn("kept the old master build but could not save its alias", "version", k.version, "err", err)
		fmt.Printf("Run `zvm alias %s %s` to name it.\n", k.alias, k.version)
		return nil
	}

	fmt.Printf("Kept the old master build as %s (alias %q)\n", k.version, k.alias)
	return nil
}
