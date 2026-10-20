//go:build !unix && !windows

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import "io/fs"

// fileIdentity is unknown on platforms without inode information (plan9,
// wasm). Dedupe still runs, but garbage collection never removes a store
// entry whose link count it cannot read.
func fileIdentity(string, fs.FileInfo) (fileID, uint64, bool) {
	return fileID{}, 0, false
}

func ownerTag(fs.FileInfo) string {
	return ""
}
