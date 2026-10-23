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

	"github.com/rogpeppe/go-internal/lockedfile"
)

// knownHashesFile is the trust-on-first-use store (similar in spirit to go.sum)
// mapping official tarball filenames to the SHA-256 of the copy that was first
// installed. It is a regular file, so GetInstalledVersions (directories only)
// and Clean (archives only) ignore it.
const knownHashesFile = "known_hashes.json"

// loadKnownHashes reads the store at path. A missing file is an empty store;
// an unreadable or corrupt one is an error, since silently resetting it would
// defeat the pinning.
func loadKnownHashes(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadKnownHashes, err)
	}

	hashes := map[string]string{}
	if err := json.Unmarshal(data, &hashes); err != nil {
		return nil, fmt.Errorf("%w: %s: %w (delete the file to reset it)", ErrBadKnownHashes, path, err)
	}
	if hashes == nil {
		return nil, fmt.Errorf("%w: %s must contain a JSON object (delete the file to reset it)", ErrBadKnownHashes, path)
	}
	return hashes, nil
}

// checkKnownHash compares sum to the hash recorded for tarName in the store at
// path. It returns recorded=false when tarName has no entry yet.
func checkKnownHash(path, tarName, sum string) (recorded bool, err error) {
	hashes, err := loadKnownHashes(path)
	if err != nil {
		return false, err
	}
	known, ok := hashes[tarName]
	if !ok {
		return false, nil
	}
	if known != sum {
		return true, fmt.Errorf("%w: %s has sha256 %s, but %s was recorded in %s when it was first installed; "+
			"if you trust the new file, remove that entry and retry", ErrKnownHashMismatch, tarName, sum, known, path)
	}
	return true, nil
}

// recordKnownHash stores sum for tarName in the store at path, replacing the
// file atomically so an interrupted write cannot corrupt existing entries.
// A persistent sidecar lock serializes the check and update across processes;
// locking the store itself would lose exclusion when it is replaced by rename.
func recordKnownHash(path, tarName, sum string) error {
	unlock, err := lockedfile.MutexAt(path + ".lock").Lock()
	if err != nil {
		return fmt.Errorf("%w: lock %s: %w", ErrBadKnownHashes, path, err)
	}
	defer unlock()

	hashes, err := loadKnownHashes(path)
	if err != nil {
		return err
	}
	if _, exists := hashes[tarName]; exists {
		// Another install may have pinned this name since the download began.
		_, err := checkKnownHash(path, tarName, sum)
		return err
	}
	hashes[tarName] = sum

	data, err := json.MarshalIndent(hashes, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), knownHashesFile+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
