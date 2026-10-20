//go:build unix

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
	"syscall"
)

// processAlive reports whether a process with the given pid exists. known is
// false when liveness cannot be determined on this platform.
func processAlive(pid int) (alive bool, known bool) {
	if pid <= 0 {
		return false, false
	}
	err := syscall.Kill(pid, 0)
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true, true
	}
	if errors.Is(err, syscall.ESRCH) {
		return false, true
	}
	return false, false
}
