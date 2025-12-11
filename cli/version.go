// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tristanisham/zvm/cli/meta"

	"github.com/charmbracelet/log"
)

// fetchVersionMap downloads the Zig version map from the configured URL.
// It parses the JSON response and writes it to a "versions.json" file in the base directory.
func (z *ZVM) fetchVersionMap() (zigVersionMap, error) {

	log.Debug("setting's VMU", "url", z.Settings.VersionMapUrl)

	req, err := http.NewRequest("GET", z.Settings.VersionMapUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "zvm "+meta.VERSION)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	versions, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rawVersionStructure := make(zigVersionMap)
	if err := json.Unmarshal(versions, &rawVersionStructure); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidVersionMap, err)
		}

		return nil, err
	}

	if err := os.WriteFile(filepath.Join(z.baseDir, "versions.json"), versions, 0755); err != nil {
		return nil, err
	}

	return rawVersionStructure, nil
}

// cleanURL removes consecutive slashes from a URL while preserving the protocol.
func cleanURL(url string) string {
	// Split the URL into two parts: protocol (e.g., "https://") and the rest
	var prefix string
	if strings.HasPrefix(url, "https://") {
		prefix = "https://"
		url = strings.TrimPrefix(url, "https://")
	} else if strings.HasPrefix(url, "http://") {
		prefix = "http://"
		url = strings.TrimPrefix(url, "http://")
	}

	// Replace multiple slashes with a single slash in the remaining part of the URL
	cleanedPath := strings.ReplaceAll(url, "//", "/")

	// Reconstruct the URL with the protocol prefix
	return prefix + cleanedPath
}

// note: the zls release-worker uses the same index format as zig, but without the latest master entry.
// fetchZlsTaggedVersionMap downloads the ZLS tagged version map from the configured URL.
// It writes the result to "versions-zls.json" in the base directory.
func (z *ZVM) fetchZlsTaggedVersionMap() (zigVersionMap, error) {
	log.Debug("setting's ZRW", "url", z.Settings.ZlsVMU)

	fullVersionMapAPI := cleanURL(z.Settings.ZlsVMU + "v1/zls/index.json")

	log.Debug("Version Map Url (95)", "func", "fetchZlsTaggedVersionMap", "url", fullVersionMapAPI)
	req, err := http.NewRequest("GET", fullVersionMapAPI, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "zvm "+meta.VERSION)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	versions, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rawVersionStructure := make(zigVersionMap)
	if err := json.Unmarshal(versions, &rawVersionStructure); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidVersionMap, err)
		}

		return nil, err
	}

	if err := os.WriteFile(filepath.Join(z.baseDir, "versions-zls.json"), versions, 0755); err != nil {
		return nil, err
	}

	return rawVersionStructure, nil
}

// note: the zls release-worker uses the same index format as zig, but without the latest master entry.
// this function does not write the result to a file.
// fetchZlsVersionByZigVersion queries the ZLS release worker for a ZLS build compatible
// with the specific Zig version and compatibility mode provided.
func (z *ZVM) fetchZlsVersionByZigVersion(version string, compatMode string) (zigVersion, error) {
	log.Debug("setting's ZRW", "url", z.Settings.ZlsVMU)

	// https://github.com/zigtools/release-worker?tab=readme-ov-file#query-parameters
	// The compatibility query parameter must be either only-runtime or full:
	//   full: Request a ZLS build that can be built and used with the given Zig version.
	//   only-runtime: Request a ZLS build that can be used at runtime with the given Zig version but may not be able to build ZLS from source.
	selectVersionUrl := cleanURL(fmt.Sprintf("%s/v1/zls/select-version?zig_version=%s&compatibility=%s", z.Settings.ZlsVMU, url.QueryEscape(version), compatMode))
	log.Debug("fetching zls version", "zigVersion", version, "url", selectVersionUrl)
	req, err := http.NewRequest("GET", selectVersionUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "zvm "+meta.VERSION)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	versions, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rawVersionStructure := make(zigVersion)
	if err := json.Unmarshal(versions, &rawVersionStructure); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidVersionMap, err)
		}

		return nil, err
	}

	if badRequest, ok := rawVersionStructure["error"].(string); ok {
		return nil, fmt.Errorf("%w: %s", ErrNoZlsVersion, badRequest)
	}

	if code, ok := rawVersionStructure["code"]; ok {
		codeStr := strconv.FormatFloat(code.(float64), 'f', 0, 64)
		msg := rawVersionStructure["message"]
		return nil, fmt.Errorf("%w: code %s: %s", ErrNoZlsVersion, codeStr, msg)
	}

	return rawVersionStructure, nil
}

// statelessFetchVersionMap is the same as fetchVersionMap but it doesn't write to disk. Will probably be depreciated and nuked from orbit when my
