//go:build !windows

// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import (
	"fmt"
	"os"
	"time"
)

// Link is a wrapper around Go's os.Symlink and os.Link functions,
// On Windows, if Link is unable to create a symlink it will attempt to create a
// hardlink before trying its automatic privilege escalation.
func Link(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}

// Relink points the link at newname to oldname, creating it if it does not
// exist. On Unix the new symlink is created beside newname and renamed over
// it, so other processes never observe a missing link. An existing entry at
// newname that is not a symlink is never replaced.
func Relink(oldname, newname string) error {
	if fi, err := os.Lstat(newname); err == nil && !IsLink(fi) {
		return &os.LinkError{Op: "relink", Old: oldname, New: newname, Err: os.ErrExist}
	}

	tmp := fmt.Sprintf("%s.tmp-%d-%d", newname, os.Getpid(), time.Now().UnixNano())
	if err := os.Symlink(oldname, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, newname); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// IsLink reports whether fi (from os.Lstat) describes a link created by Link.
func IsLink(fi os.FileInfo) bool {
	return fi.Mode()&os.ModeSymlink != 0
}
