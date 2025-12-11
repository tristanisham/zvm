// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
)

var (
	// ErrMissingBundlePath is returned when the download path for a version's bundle cannot be found.
	ErrMissingBundlePath = errors.New("bundle download path not found")
	// ErrUnsupportedSystem is returned when the current operating system or architecture is not supported by Zig.
	ErrUnsupportedSystem = errors.New("unsupported system for Zig")
	// ErrUnsupportedVersion is returned when the requested Zig version is not available.
	ErrUnsupportedVersion = errors.New("unsupported Zig version")
	// ErrMissingInstallPathEnv is returned when the ZVM_INSTALL environment variable is missing.
	ErrMissingInstallPathEnv = errors.New("env 'ZVM_INSTALL' is not set")
	// ErrFailedUpgrade is returned when the self-upgrade process fails.
	ErrFailedUpgrade = errors.New("failed to self-upgrade zvm")
	// ErrInvalidVersionMap is returned when the fetched version map JSON is invalid or corrupted.
	ErrInvalidVersionMap = errors.New("invalid version map format")
	// ErrInvalidInput is returned when the user provides invalid input arguments.
	ErrInvalidInput = errors.New("invalid input")
	// ErrDownloadFail is returned when fetching Zig, or constructing a target URL to fetch Zig, fails.
	ErrDownloadFail = errors.New("failed to download Zig")
	// ErrNoZlsVersion is returned when the ZLS release worker returns an error or no version is found.
	ErrNoZlsVersion = errors.New("zls release worker returned error")
	// ErrMissingVersionInfo is returned when version information is missing from the API response.
	ErrMissingVersionInfo = errors.New("version info not found")
	// ErrMissingShasum is returned when the SHA256 checksum is missing for a download.
	ErrMissingShasum = errors.New("shasum not found")
	// ErrZigNotInstalled is returned when the `zig` executable cannot be found in the PATH.
	ErrZigNotInstalled = errors.New("exec `zig` not found on $PATH")
)
