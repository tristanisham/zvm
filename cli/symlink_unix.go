//go:build !windows
package cli

func newSymlink(oldname, newname string) error {
	_ = oldname
	_ = newname
	return nil
}