//go:build !windows

package meta

import "os"

func Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}
