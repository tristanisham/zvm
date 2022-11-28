package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func Install(version string) error {
	homedir, err := os.UserHomeDir()
	if err != nil {
		homedir = "~"
	}
	zvm := filepath.Join(homedir, ".zvm")
	os.Mkdir(zvm, 0755) 

	req, err := http.NewRequest("GET", "https://ziglang.org/download/index.json", nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "zvm (Zig Version Manager) 0.0.2")
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	versions, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(zvm, "versions.json"), versions, 0755); err != nil {
		return err
	}

	rawVersionStructure := make (map[any]any)
	if err := json.Unmarshal(versions, &rawVersionStructure); err != nil {
		return err
	}


	return installVersion(rawVersionStructure)
}

func installVersion(data map[any]any) error {
	return nil
}