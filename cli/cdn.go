package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"zvm/cli/meta"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

type CDNResponse struct {
	DownloadUrl string `json:"downloadUrl"`
}

// doWeHaveIt checks if a zig version exists in the third-party zig store.
// If yes, it will return the download link
func (z *ZVM) doWeHaveIt(version string) (string, bool) {
	// Parse the version repo URL from settings
	target, err := url.Parse(z.Settings.VersionRepo)
	if err != nil {
		log.Error(err)
		return "", false
	}

	zArch, zOs := zigStyleSysInfo()

	// Add a unique ID to the query parameters
	uid := uuid.New()
	queryVals := target.Query()
	queryVals.Add("id", uid.String())

	// Add version to the query parameters
	queryVals.Add("version", version)
	queryVals.Add("os", zOs)
	queryVals.Add("arch", zArch)
	// Set the modified query parameters back to the target URL
	target.RawQuery = queryVals.Encode()

	// Create a new http client
	req, err := http.NewRequest(http.MethodGet, target.String(), nil)
	if err != nil {
		log.Error(err)
		return "", false
	}

	req.Header.Set("User-Agent", fmt.Sprintf("ZVM/%s (%s; %s)", meta.VERSION, runtime.GOOS, runtime.GOARCH))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err)
		return "", false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return "", false
	}

	var info CDNResponse
	if err := json.Unmarshal(body, &info); err != nil {
		log.Error(err)
		return "", false
	}

	// For now, I'm returning the modified URL for demonstration
	return info.DownloadUrl, false
}
