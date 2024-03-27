//go:build !linux

// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import "os"

func canModifyFile(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	// Get the file's permission mode
	perm := fileInfo.Mode().Perm()

	// Check if the file is writable by the current user
	if perm&0200 != 0 {
		return true, nil
	}

	// if runtime.GOOS == "linux" {
	// 	// If the file isn't globally writable, check if it's writable by the file's group
	// 	if perm&0020 != 0 {
	// 		currentUser, err := user.Current()
	// 		if err != nil {
	// 			return false, err
	// 		}
	// 		fileGroup, err := user.LookupGroupId(fmt.Sprint(fileInfo.Sys().(*syscall.Stat_t).Gid))
	// 		if err != nil {
	// 			return false, err
	// 		}
	// 		if currentUser.Gid == fileGroup.Gid {
	// 			return true, nil
	// 		}
	// 	}
	// }

	return false, nil
}
