// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	"golang.org/x/mod/semver"
)

// resolveVersionShorthand takes a possibly-abbreviated version string and a
// list of available version strings, and returns the best matching full version.
// Examples:
//
//	"0.12"  + ["0.12.0", "0.13.0"]  -> "0.12.0"
//	".12"   + ["0.12.0", "0.13.0"]  -> "0.12.0"
//	"0.15"  + ["0.15.0", "0.15.1"]  -> "0.15.1" (latest patch)
//	"0.12.0" + ["0.12.0"]           -> "0.12.0" (exact, unchanged)
func resolveVersionShorthand(input string, available []string) (string, error) {
	log.Debug("resolveVersionShorthand", "input", input, "candidates", len(available))

	if input == "" {
		return "", fmt.Errorf("empty version string")
	}

	// Normalize: ".12" -> "0.12"
	if strings.HasPrefix(input, ".") {
		normalized := "0" + input
		log.Debug("normalized dot prefix", "from", input, "to", normalized)
		input = normalized
	}

	// Special non-semver names pass through unchanged
	if isSpecialVersion(input) {
		log.Debug("special version passthrough", "version", input)
		return input, nil
	}

	// "stable" resolves to the highest non-dev, non-master release
	if input == "stable" {
		log.Debug("resolving stable alias")
		return resolveStable(available)
	}

	// Exact match — return immediately
	if slices.Contains(available, input) {
		log.Debug("exact version match", "version", input)
		return input, nil
	}

	// Prefix match: "0.12" matches "0.12.0", "0.12.1", etc.
	// Using input+"." prevents "0.1" from matching "0.13.0"
	var matches []string
	prefix := input + "."
	for _, v := range available {
		if isSpecialVersion(v) {
			continue
		}
		if strings.HasPrefix(v, prefix) {
			matches = append(matches, v)
		}
	}

	if len(matches) == 0 {
		log.Debug("no prefix matches found", "input", input)
		return input, nil
	}

	resolved := latestVersion(matches)
	log.Debug("prefix match resolved", "input", input, "matches", matches, "resolved", resolved)
	return resolved, nil
}

// isSpecialVersion returns true for non-semver version identifiers
// that should not be resolved via prefix matching.
func isSpecialVersion(v string) bool {
	switch v {
	case "master", "latest":
		return true
	}
	return false
}

// resolveStable returns the highest non-dev, non-master release version.
func resolveStable(available []string) (string, error) {
	var releases []string
	for _, v := range available {
		if isSpecialVersion(v) {
			continue
		}
		if strings.Contains(v, "-dev") {
			log.Debug("resolveStable: skipping dev version", "version", v)
			continue
		}
		// Must be valid semver
		if !semver.IsValid("v" + v) {
			log.Debug("resolveStable: skipping invalid semver", "version", v)
			continue
		}
		releases = append(releases, v)
	}
	if len(releases) == 0 {
		log.Debug("resolveStable: no stable releases found")
		return "", fmt.Errorf("no stable release found")
	}
	stable := latestVersion(releases)
	log.Debug("resolveStable", "stable", stable, "releaseCount", len(releases))
	return stable, nil
}

// latestVersion returns the highest semver version from a non-empty slice.
// Version strings are expected without a "v" prefix (e.g. "0.12.0").
func latestVersion(versions []string) string {
	best := versions[0]
	for _, v := range versions[1:] {
		if semver.Compare("v"+v, "v"+best) > 0 {
			best = v
		}
	}
	return best
}
