package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/charmbracelet/log"
)

const VERSION = "v0.2.2"

func (z *ZVM) fetchVersionMap() (map[string]ZigRelease, error) {

	req, err := http.NewRequest("GET", "https://ziglang.org/download/index.json", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "zvm (Zig Version Manager) "+VERSION)
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

	// Unmarshal into a map
	rawData := make(map[string]map[string]json.RawMessage)
	err = json.Unmarshal(versions, &rawData)
	if err != nil {
		log.Fatal(err)
	}

	result := make(map[string]ZigRelease)

	for version, details := range rawData {

		// Create and fill the ZigRelease struct
		release := ZigRelease{
			Releases: make(map[string]tarball),
		}

		for key, value := range details {
			switch key {
			case "version", "date", "docs", "stdDocs", "notes":
				// For these keys, unmarshal the value into the corresponding field in ZigRelease
				field := reflect.ValueOf(&release).Elem().FieldByName(strings.Title(key))
				err := json.Unmarshal(value, field.Addr().Interface())
				if err != nil {
					log.Fatalf("failed to unmarshal key %q: %v", key, err)
				}
			default:
				// For other keys, unmarshal the value into a tarball and put it in the map
				var t tarball
				err := json.Unmarshal(value, &t)
				if err != nil {
					log.Fatalf("failed to unmarshal tarball %q: %v", key, err)
				}
				release.Releases[key] = t
			}
		}

		result[version] = release
	}

	// rawVersionStructure := make(map[string]ZigRelease)
	// if err := json.Unmarshal(versions, &rawVersionStructure); err != nil {
	// 	return nil, err
	// }

	return result, nil
}

// statelessFetchVersionMap is the same as fetchVersionMap but it doesn't write to disk. Will probably be depreciated and nuked from orbit when my
