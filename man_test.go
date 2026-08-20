// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManPagesCoverCommandsAndFlags(t *testing.T) {
	rootPage := readManPage(t, "zvm")
	if !strings.Contains(rootPage, "--color") {
		t.Error("zvm(1) does not document --color")
	}

	for _, command := range zvmApp.Commands {
		if command.Hidden || command.Name == "help" {
			continue
		}

		pageName := "zvm-" + command.Name
		page := readManPage(t, pageName)
		if !strings.Contains(rootPage, pageName+" (1)") {
			t.Errorf("zvm(1) does not link to %s(1)", pageName)
		}

		for _, flag := range command.Flags {
			for _, name := range flag.Names() {
				if name == "help" || name == "h" {
					continue
				}
				prefix := "--"
				if len(name) == 1 {
					prefix = "-"
				}
				if !strings.Contains(page, prefix+name) {
					t.Errorf("%s(1) does not document %s%s", pageName, prefix, name)
				}
			}
		}
	}

	completionPage := readManPage(t, "zvm-completion")
	for _, shell := range []string{"bash", "zsh", "fish", "pwsh"} {
		if !strings.Contains(completionPage, ".B "+shell) {
			t.Errorf("zvm-completion(1) does not document %s", shell)
		}
	}
}

func readManPage(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("man", name+".1")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}
