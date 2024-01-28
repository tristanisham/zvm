package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"zvm/cli/meta"
)

func (z *ZVM) fetchVersionMap() (zigVersionMap, error) {

	defaultVersionMapUrl := "https://ziglang.org/download/index.json"
	versionMapUrl := z.Settings.VersionMapUrl
	if len(versionMapUrl) == 0 {
		versionMapUrl = defaultVersionMapUrl
	}

	req, err := http.NewRequest("GET", versionMapUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "zvm "+meta.VERSION)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	versions, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(z.zvmBaseDir, "versions.json"), versions, 0755); err != nil {
		return nil, err
	}

	rawVersionStructure := make(zigVersionMap)
	if err := json.Unmarshal(versions, &rawVersionStructure); err != nil {
		return nil, err
	}

	return rawVersionStructure, nil
}

// statelessFetchVersionMap is the same as fetchVersionMap but it doesn't write to disk. Will probably be depreciated and nuked from orbit when my
