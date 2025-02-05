//go:build darwin || plan9

package cli

import (
	"os"
	"path/filepath"
)

func zvmDataDirPath(home string) string {
	zvm_path := os.Getenv("ZVM_PATH")
	if zvm_path == "" {
		zvm_path = filepath.Join(home, ".zvm")
	}
	return zvm_path
}

func zvmStateDirPath(home string) string {
	return zvmDataDirPath(home)
}
func zvmConfigDirPath(home string) string {
	return zvmDataDirPath(home)
}
func zvmBinDirPath(home string) string {
	return zvmDataDirPath(home)
}
func zvmCacheDirPath(home string) string {
	return zvmDataDirPath(home)
}
