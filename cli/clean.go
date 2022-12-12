package cli

import (
	"os"
	"path/filepath"
	"strings"
)

func (z *ZVM) Clean() error {
	dir, err := os.ReadDir(z.zvmBaseDir)
	if err != nil {
		return err
	}

	for _, entry := range dir {
		if strings.Contains(entry.Name(), "bin") {
			continue
		}

		if filepath.Ext(entry.Name()) == ".zip" || filepath.Ext(entry.Name()) == ".xz" {
			if err := os.Remove(filepath.Join(z.zvmBaseDir, entry.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}
