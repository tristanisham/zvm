package cli

import (
	"os"
	"path/filepath"
)

func (z *ZVM) Use(ver string) error {
	z.loadVersionCache()
	if err := z.getVersion(ver); err == nil {
		return z.Install(ver)
	}
	return z.setBin(ver)
}

func (z *ZVM) setBin(ver string) error {
	version_path := filepath.Join(z.zvmBaseDir, "bin")
	if _, err := os.Stat(version_path); os.IsExist(err) {
		if err := os.Remove(filepath.Join(z.zvmBaseDir, "bin")); err != nil {
			return err
		}
	}

	if err := os.Symlink(version_path, "bin"); err != nil {
		return err
	}
	return nil
}
