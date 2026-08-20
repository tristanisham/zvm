//go:build !windows && !darwin

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import "os"

// browserCommands returns the platform's URL handlers, most preferred first.
// Without a display server there is nothing for a browser to open onto, so an
// empty list sends the caller straight to printing the URL.
func browserCommands() [][]string {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return nil
	}

	return [][]string{{"xdg-open"}, {"www-browser"}, {"x-www-browser"}}
}
