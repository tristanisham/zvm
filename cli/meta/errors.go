// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import "errors"

var (
	ErrWinEscToAdmin    = errors.New("unable to rerun as Windows Administrator")
	ErrEscalatedSymlink = errors.New("unable to symlink as Administrator")
)
