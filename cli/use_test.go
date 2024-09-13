package cli

import (
	"os"
	"testing"
)

func TestSymlinkExists(t *testing.T) {
	if err := os.Symlink("use_test.go", "symlink.test"); err != nil {
		t.Error(err)
	}

	stat, err := os.Lstat("symlink.test")
	if err != nil {
		t.Errorf("%q: %s", err, stat.Name())
	}

	defer os.Remove("symlink.test")
}
