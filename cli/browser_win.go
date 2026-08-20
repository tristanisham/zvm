//go:build windows

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

// browserCommands returns the platform's URL handlers, most preferred first.
// "start" is a cmd builtin, and its first quoted argument is taken as a window
// title, so the empty string reserves that slot for the URL.
func browserCommands() [][]string {
	return [][]string{{"cmd", "/c", "start", ""}}
}
