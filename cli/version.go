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

	"strings"

	"github.com/tristanisham/zvm/cli/meta"

	"github.com/charmbracelet/log"
)

// fetchVersionMap downloads the Zig version map from the configured URL.
// It parses the JSON response and writes it to a "versions.json" file in the base directory.
func (z *ZVM) fetchVersionMap() (zigVersionMap, error) {
	log.Debug("setting's VMU", "url", z.Settings.VersionMapUrl)
	return z.fetchAndSaveVersionMap(z.Settings.VersionMapUrl, "versions.json")
}

const maxVersionMapSize = 32 << 20

// versionHTTPError reports a non-200 response from a version endpoint.
// Message holds the server's JSON "error" field when it sent one.
type versionHTTPError struct {
	Status  string
	Message string
}

func (e *versionHTTPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("version request failed: %s: %s", e.Status, e.Message)
	}
	return "version request failed: " + e.Status
}

// fetchVersionJSON bounds network time and response size for all version endpoints.
func fetchVersionJSON(endpoint string, result any) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "zvm "+meta.VERSION)
	client := &http.Client{Timeout: downloadTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		httpErr := &versionHTTPError{Status: resp.Status}
		var body struct {
			Error string `json:"error"`
		}
		if data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10)); err == nil && json.Unmarshal(data, &body) == nil {
			httpErr.Message = body.Error
		}
		return nil, httpErr
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxVersionMapSize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxVersionMapSize {
		return nil, fmt.Errorf("%w: response exceeds %d bytes", ErrInvalidVersionMap, maxVersionMapSize)
	}
	if err := json.Unmarshal(body, result); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidVersionMap, err)
		}
		return nil, err
	}
	return body, nil
}

func (z *ZVM) fetchAndSaveVersionMap(endpoint, filename string) (zigVersionMap, error) {
	versions := make(zigVersionMap)
	body, err := fetchVersionJSON(endpoint, &versions)
	if err != nil {
		return nil, err
	}
	file := filepath.Join(z.baseDir, filename)
	if err := os.WriteFile(file, body, 0o644); err != nil {
		return nil, err
	}
	if err := os.Chmod(file, 0o644); err != nil {
		return nil, err
	}
	return versions, nil
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
	endpoint := cleanURL(z.Settings.ZlsVMU + "v1/zls/index.json")
	log.Debug("fetching zls version map", "url", endpoint)
	return z.fetchAndSaveVersionMap(endpoint, "versions-zls.json")
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
	rawVersionStructure := make(zigVersion)
	if _, err := fetchVersionJSON(selectVersionUrl, &rawVersionStructure); err != nil {
		// The release worker answers invalid queries with 400 {"error": "..."}.
		var httpErr *versionHTTPError
		if errors.As(err, &httpErr) && httpErr.Message != "" {
			return nil, fmt.Errorf("%w: %s", ErrNoZlsVersion, httpErr.Message)
		}
		return nil, err
	}

	if badRequest, ok := rawVersionStructure["error"].(string); ok {
		return nil, fmt.Errorf("%w: %s", ErrNoZlsVersion, badRequest)
	}

	if code, ok := rawVersionStructure["code"]; ok {
		codeStr := fmt.Sprint(code)
		msg := rawVersionStructure["message"]
		return nil, fmt.Errorf("%w: code %s: %s", ErrNoZlsVersion, codeStr, msg)
	}

	return rawVersionStructure, nil
}
