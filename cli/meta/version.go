// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import (
	"fmt"
	"os"
	"runtime"
)

const (
	VERSION = "v0.8.24"

	// VERSION = "v0.0.0" // For testing zvm upgrade

)

var VerCopy = fmt.Sprintf("%s %s/%s", VERSION, runtime.GOOS, runtime.GOARCH)

var Debug bool

func init() {
	if _, ok := os.LookupEnv("ZVM_DEBUG"); ok {
		Debug = true
	}
}
