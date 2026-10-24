// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanisham/zvm/cli/meta"
	opts "github.com/urfave/cli/v3"
)

func TestManPagesCoverCommandsAndFlags(t *testing.T) {
	rootPage := readManPage(t, "zvm")
	assertManFlags(t, "zvm", rootPage, zvmApp.Flags)

	for _, command := range zvmApp.Commands {
		if command.Hidden || command.Name == "help" {
			continue
		}

		pageName := "zvm-" + command.Name
		page := readManPage(t, pageName)
		if !strings.Contains(rootPage, pageName+" (1)") {
			t.Errorf("zvm(1) does not link to %s(1)", pageName)
		}

		assertManCommand(t, pageName, page, command)
	}

	completionPage := readManPage(t, "zvm-completion")
	for _, shell := range []string{"bash", "zsh", "fish", "pwsh"} {
		if !strings.Contains(completionPage, ".B "+shell) {
			t.Errorf("zvm-completion(1) does not document %s", shell)
		}
	}
}

func assertManCommand(t *testing.T, pageName, page string, command *opts.Command) {
	t.Helper()

	assertManFlags(t, pageName, page, command.Flags)
	for _, name := range command.Names() {
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(page) {
			t.Errorf("%s(1) does not document command name or alias %s", pageName, name)
		}
	}
	for _, sub := range command.Commands {
		if !sub.Hidden && sub.Name != "help" {
			assertManCommand(t, pageName, page, sub)
		}
	}
}

func assertManFlags(t *testing.T, pageName, page string, flags []opts.Flag) {
	t.Helper()

	for _, flag := range flags {
		for _, name := range flag.Names() {
			if name == "help" || name == "h" {
				continue
			}
			prefix := "--"
			if len(name) == 1 {
				prefix = "-"
			}
			pattern := `(^|[\s"])` + regexp.QuoteMeta(prefix+name) + `([\s",]|$)`
			if !regexp.MustCompile(pattern).MatchString(page) {
				t.Errorf("%s(1) does not document %s%s", pageName, prefix, name)
			}
		}
	}
}

func TestManPageVersions(t *testing.T) {
	pages, err := filepath.Glob(filepath.Join("man", "*.1"))
	if err != nil || len(pages) == 0 {
		t.Fatalf("find manpages: %v (found %d)", err, len(pages))
	}

	for _, path := range pages {
		name := strings.TrimSuffix(filepath.Base(path), ".1")
		t.Run(name, func(t *testing.T) {
			page := readManPage(t, name)
			pattern := `(?m)^\.TH .* "ZVM ` + regexp.QuoteMeta(meta.VERSION) + `" "User Commands"$`
			if !regexp.MustCompile(pattern).MatchString(page) {
				t.Errorf("%s(1) release header must match %s in cli/meta/version.go", name, meta.VERSION)
			}
		})
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
