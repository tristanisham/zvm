package cli

import (
	"encoding/json"
	// "fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"zvm/cli/meta"

	"github.com/charmbracelet/log"
	// "github.com/tristanisham/clr"
)

func (z *ZVM) fetchVersionMap() (zigVersionMap, error) {

	log.Debug("inital VMU", "url", z.Settings.VersionMapUrl)

	if err := z.loadSettings(); err != nil {
		log.Warnf("could not read settings: %q", err)
		log.Debug("vmu", z.Settings.VersionMapUrl)
	}

	defaultVersionMapUrl := "https://ziglang.org/download/index.json"

	versionMapUrl := z.Settings.VersionMapUrl

	log.Debug("setting's VMU", "url", versionMapUrl)

	if len(versionMapUrl) == 0 {
		versionMapUrl = defaultVersionMapUrl
	}

	// // Limited warning until I get to properly test this code.
	// if versionMapUrl != defaultVersionMapUrl {
	// 	fmt.Println("This command is currently in beta and may break your install.")
	// 	fmt.Printf("To reset your version map, run %s", clr.Green("zvm -unstable-vmu default"))
	// }

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
