// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"bufio"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

// VersionSource describes where a project's Zig version was found.
type VersionSource struct {
	// Version is the requested Zig version, with any leading "v" removed.
	Version string
	// File is the path of the file the version was read from.
	File string
	// Key is "zvm-lock" (build.zig) or "minimum_zig_version" (build.zig.zon).
	Key string
}

const (
	versionKeyLock    = "zvm-lock"
	versionKeyMinimum = "minimum_zig_version"
)

// FindVersionSource looks for a Zig version request in start and then in each
// of its parents, returning the nearest one. Within a single directory a
// `//! zvm-lock: <version>` line in build.zig takes precedence over
// `.minimum_zig_version` in build.zig.zon. A directory whose files exist but
// declare no version is skipped, so the search continues upward.
//
// ok is false when no directory up to the filesystem root declares a version.
func FindVersionSource(start string) (src VersionSource, ok bool, err error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return VersionSource{}, false, err
	}

	for {
		if src, ok := versionSourceIn(dir); ok {
			return src, true, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return VersionSource{}, false, nil
		}
		dir = parent
	}
}

// versionSourceIn checks a single directory for a version request.
func versionSourceIn(dir string) (VersionSource, bool) {
	buildZig := filepath.Join(dir, "build.zig")
	if data, err := os.ReadFile(buildZig); err == nil {
		if ver, ok := parseZvmLock(data); ok {
			return VersionSource{Version: ver, File: buildZig, Key: versionKeyLock}, true
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		log.Debug("reading build.zig", "path", buildZig, "err", err)
	}

	buildZon := filepath.Join(dir, "build.zig.zon")
	if data, err := os.ReadFile(buildZon); err == nil {
		if ver, ok := parseMinimumZigVersion(data); ok {
			return VersionSource{Version: ver, File: buildZon, Key: versionKeyMinimum}, true
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		log.Debug("reading build.zig.zon", "path", buildZon, "err", err)
	}

	return VersionSource{}, false
}

// parseZvmLock returns the version from the first `//! zvm-lock: <version>`
// line in a build.zig file. It uses the same key/value form as Sync.
func parseZvmLock(data []byte) (string, bool) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		rest, found := strings.CutPrefix(line, "//!")
		if !found {
			continue
		}

		key, value, found := strings.Cut(rest, ":")
		if !found || !strings.EqualFold(strings.TrimSpace(key), versionKeyLock) {
			continue
		}

		value = strings.TrimPrefix(strings.TrimSpace(value), "v")
		if value == "" || strings.ContainsAny(value, " \t:") {
			log.Debug("ignoring malformed zvm-lock", "line", line)
			continue
		}
		return value, true
	}
	return "", false
}

// parseMinimumZigVersion returns the uncommented `.minimum_zig_version` value
// from a build.zig.zon file.
func parseMinimumZigVersion(data []byte) (string, bool) {
	matches := minZigVersionRe.FindSubmatch(data)
	if len(matches) < 2 {
		return "", false
	}
	return strings.TrimPrefix(string(matches[1]), "v"), true
}
