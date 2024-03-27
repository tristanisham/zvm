//go:build !windows

// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import "os"

func Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}
