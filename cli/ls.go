package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

func (z *ZVM) ListVersions() error {
	iv, err := z.installVersions()
	if err != nil {
		return err
	}

	var versions string
	for key := range *iv {
		versions += fmt.Sprintf("%s\n", key)
	}

	fmt.Print(versions)

	return nil
}

func (z ZVM) installVersions() (*map[string]string, error) {
	dir, err := os.ReadDir(z.zvmBaseDir)
	if err != nil {
		return nil, err
	}

	if err := z.loadVersionCache(); err != nil {
		return nil, err
	}

	result := make(map[string]string)

	for _, entry := range dir {
		if !entry.IsDir() {
			continue
		}

		if _, ok := z.zigVersions[entry.Name()]; ok {
			result[entry.Name()] = filepath.Join(z.zvmBaseDir, entry.Name())
		}
	}

	return &result, nil
}
