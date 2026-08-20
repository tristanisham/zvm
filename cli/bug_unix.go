//go:build !windows

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"os"
	"runtime"
	"strings"
)

// bugOSDetails describes the host beyond GOOS/GOARCH: kernel release on every
// Unix-like system, plus the distribution or macOS release where one is
// discoverable.
func bugOSDetails() string {
	var b strings.Builder

	b.WriteString(bugCommandOutput("uname -srm", "uname", "-srm"))

	switch runtime.GOOS {
	case "darwin":
		b.WriteString(bugCommandOutput("sw_vers", "sw_vers", "-productVersion"))
	case "linux":
		if name := linuxDistribution(); name != "" {
			b.WriteString("distribution: " + name + "\n")
		}
	}

	return b.String()
}

// linuxDistribution reads PRETTY_NAME out of os-release. An unreadable or
// absent file reports nothing rather than failing the report.
func linuxDistribution() string {
	for _, path := range []string{"/etc/os-release", "/usr/lib/os-release"} {
		contents, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		for line := range strings.Lines(string(contents)) {
			name, value, found := strings.Cut(strings.TrimSpace(line), "=")
			if found && name == "PRETTY_NAME" {
				return strings.Trim(value, `"`)
			}
		}
	}

	return ""
}
