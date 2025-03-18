//go:build !windows

// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import "os"

// Link is a wrapper around Go's os.Symlink and os.Link functions,
// On Windows, if Link is unable to create a symlink it will attempt to create a
// hardlink before trying its automatic privilege escalation.
func Link(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}
