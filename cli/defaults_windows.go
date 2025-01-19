// +build windows

package cli

import (
	"os"
	"path/filepath"
)

func ZvmDataDirPath(home string) string {
	zvm_path := os.Getenv("ZVM_PATH")
	if zvm_path == "" {
		zvm_path = filepath.Join(home, ".zvm")
	}
	return zvm_path
}

func ZvmStateDirPath(home string) string {
	return ZvmDataDirPath(home)
}
func ZvmConfigDirPath(home string) string {
	return ZvmDataDirPath(home)
}
func ZvmBinDirPath(home string) string {
	return ZvmDataDirPath(home)
}
func ZvmCacheDirPath(home string) string {
	return ZvmDataDirPath(home)
}
