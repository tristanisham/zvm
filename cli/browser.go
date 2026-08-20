// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"os"
	"os/exec"

	"github.com/charmbracelet/log"
)

// openBrowser tries to open url in the user's browser, reporting whether one
// of the candidate commands started successfully. $BROWSER wins over the
// platform default, matching the convention other Unix tooling follows.
func openBrowser(url string) bool {
	candidates := make([][]string, 0, 4)
	if browser := os.Getenv("BROWSER"); browser != "" {
		candidates = append(candidates, []string{browser})
	}
	candidates = append(candidates, browserCommands()...)

	for _, candidate := range candidates {
		cmd := exec.Command(candidate[0], append(candidate[1:], url)...)
		if err := cmd.Start(); err != nil {
			log.Debug("browser: launch failed", "command", candidate[0], "error", err)
			continue
		}

		// The launcher is expected to hand off to an already-running browser
		// and exit; reaping it here keeps zvm from leaving a zombie behind.
		go func() {
			if err := cmd.Wait(); err != nil {
				log.Debug("browser: exited with error", "command", candidate[0], "error", err)
			}
		}()

		return true
	}

	return false
}
