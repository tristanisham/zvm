package cli

import (
	"encoding/json"
	"io"
	"net/http"
)

const VERSION = "v0.2.0"

func (z *ZVM) fetchVersionMap() (zigVersionMap, error) {

	req, err := http.NewRequest("GET", "https://ziglang.org/download/index.json", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "zvm (Zig Version Manager) " + VERSION)
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

	rawVersionStructure := make(zigVersionMap)
	if err := json.Unmarshal(versions, &rawVersionStructure); err != nil {
		return nil, err
	}

	return rawVersionStructure, nil
}
