//go:build !windows

// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import "os"

// Symlink is a wrapper around Go's os.Symlink,
// but with automatic privilege escalation on windows
// for systems that do not support non-admin symlinks.
func Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}
