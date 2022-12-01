package cli

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

func Initialize() *ZVM {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	zvm_path := filepath.Join(home, ".zvm")
	if _, err := os.Stat(zvm_path); os.IsNotExist(err) {
		if err := os.MkdirAll(zvm_path, 0775); err != nil {
			log.Fatal(err)
		}
	}

	return &ZVM{
		zvmBaseDir: zvm_path,
	}
}

type ZVM struct {
	zvmBaseDir  string
	zigVersions zigVersionMap
}

// A representaiton of the offical json schema for Zig versions
type zigVersionMap = map[string]zigVersion

// A representation of individual Zig versions
type zigVersion = map[string]any

func (z *ZVM) loadVersionCache() error {
	ver, err := os.ReadFile(filepath.Join(z.zvmBaseDir, "versions.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(ver, &z.zigVersions); err != nil {
		return err
	}
	return nil
}
