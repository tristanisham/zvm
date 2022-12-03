package cli

import (
	"log"
	"os"
	"path/filepath"

	"github.com/tristanisham/clr"
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
	if err := os.Remove(filepath.Join(z.zvmBaseDir, "bin")); err != nil {
		log.Println(clr.Yellow(err))
	}

	if err := os.Symlink(version_path, filepath.Join(z.zvmBaseDir, "bin")); err != nil {
		return err
	}
	return nil
}
