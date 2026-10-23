//go:build !unix && !windows

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

// processAlive cannot determine liveness on this platform (e.g. Plan 9), so
// per-shell link pruning falls back to age alone.
func processAlive(pid int) (alive bool, known bool) {
	return false, false
}
