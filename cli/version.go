// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"encoding/json"
	"errors"
	"fmt"

	// "fmt"
	"github.com/tristanisham/zvm/cli/meta"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	// "github.com/tristanisham/clr"
)

func (z *ZVM) fetchVersionMap() (zigVersionMap, error) {

	log.Debug("inital VMU", "url", z.Settings.VersionMapUrl)

	if err := z.loadSettings(); err != nil {
		log.Warnf("could not load version map from settings: %q", err)
		log.Debug("vmu", z.Settings.VersionMapUrl)
	}

	defaultVersionMapUrl := "https://ziglang.org/download/index.json"

	versionMapUrl := z.Settings.VersionMapUrl

	log.Debug("setting's VMU", "url", versionMapUrl)

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

	rawVersionStructure := make(zigVersionMap)
	if err := json.Unmarshal(versions, &rawVersionStructure); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidVersionMap, err)
		}

		return nil, err
	}

	if err := os.WriteFile(filepath.Join(z.baseDir, "versions.json"), versions, 0755); err != nil {
		return nil, err
	}

	return rawVersionStructure, nil
}

// statelessFetchVersionMap is the same as fetchVersionMap but it doesn't write to disk. Will probably be depreciated and nuked from orbit when my
