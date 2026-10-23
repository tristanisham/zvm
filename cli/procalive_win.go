//go:build windows

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"errors"

	"golang.org/x/sys/windows"
)

// stillActive is the exit code GetExitCodeProcess reports for a running process.
const stillActive = 259

// processAlive reports whether a process with the given pid exists. known is
// false when liveness cannot be determined.
func processAlive(pid int) (alive bool, known bool) {
	if pid <= 0 {
		return false, false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return true, true
		}
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			// OpenProcess reports a pid that does not exist this way.
			return false, true
		}
		return false, false
	}
	defer windows.CloseHandle(h)

	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false, false
	}
	return code == stillActive, true
}
