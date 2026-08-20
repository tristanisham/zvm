//go:build darwin

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

// browserCommands returns the platform's URL handlers, most preferred first.
func browserCommands() [][]string {
	return [][]string{{"/usr/bin/open"}}
}
