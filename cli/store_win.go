//go:build windows

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"io/fs"
	"syscall"
)

// fileIdentity reports which NTFS file record path refers to and how many
// hard links point at it. Windows' os.FileInfo does not expose this without
// opening the file, so it is queried by handle, the same way os.SameFile does.
func fileIdentity(path string, _ fs.FileInfo) (id fileID, nlink uint64, ok bool) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return fileID{}, 0, false
	}

	// Access 0 queries attributes only; share everything so this never blocks
	// (or is blocked by) a running zig.exe.
	h, err := syscall.CreateFile(p, 0,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return fileID{}, 0, false
	}
	defer syscall.CloseHandle(h)

	var d syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(h, &d); err != nil {
		return fileID{}, 0, false
	}

	return fileID{
		dev: uint64(d.VolumeSerialNumber),
		ino: uint64(d.FileIndexHigh)<<32 | uint64(d.FileIndexLow),
	}, uint64(d.NumberOfLinks), true
}

// ownerTag is empty on Windows: files under %USERPROFILE%\.zvm inherit the
// directory's ACL, and Go exposes no owner in os.FileInfo.
func ownerTag(fs.FileInfo) string {
	return ""
}
