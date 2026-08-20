//go:build windows

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

// bugOSDetails describes the host beyond GOOS/GOARCH. "ver" is a cmd builtin,
// so it has to be run through the shell rather than executed directly.
func bugOSDetails() string {
	return bugCommandOutput("cmd /c ver", "cmd", "/c", "ver")
}
