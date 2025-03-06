// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import (
	"fmt"
	"runtime"
)

const (
	VERSION = "v0.8.5"
)

var ForceUpgrade = false // for testing Upgrade command

var VerCopy = fmt.Sprintf("%s %s/%s", VERSION, runtime.GOOS, runtime.GOARCH)
