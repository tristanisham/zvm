// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import (
	"fmt"
	"runtime"
)

const (
	VERSION = "v0.7.0"
	// VERSION = "v0.0.0" // For testing zvm upgrade

)


var (
	VerCopy = fmt.Sprintf("zvm %s %s/%s", VERSION, runtime.GOOS, runtime.GOARCH)
)

