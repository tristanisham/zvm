//go:build unix

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"io/fs"
	"syscall"
)

// fileIdentity reports which inode info describes and how many hard links
// point at it. info must come from os.Lstat/os.Stat (or a DirEntry's Info),
// which on Unix carries a *syscall.Stat_t; path is unused here.
func fileIdentity(_ string, info fs.FileInfo) (id fileID, nlink uint64, ok bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileID{}, 0, false
	}
	// Field widths differ by platform (Dev is int32 on darwin, Nlink uint16),
	// so widen everything explicitly.
	return fileID{dev: uint64(st.Dev), ino: uint64(st.Ino)}, uint64(st.Nlink), true
}

// ownerTag returns the uid/gid of info as part of the store key. Hard links
// share their owner, so files owned by different users must never be merged.
func ownerTag(info fs.FileInfo) string {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d.%d", st.Uid, st.Gid)
}
