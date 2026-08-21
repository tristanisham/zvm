// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateSymlinkSkipUse(t *testing.T) {
	z := &ZVM{baseDir: t.TempDir()}

	z.createSymlink("0.15.1", true)

	if _, err := os.Lstat(filepath.Join(z.baseDir, "bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("bin link exists after skip-use install: %v", err)
	}
}
