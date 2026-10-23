// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseUploadRequiresDraft(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release workflow runs on Linux")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is unavailable")
	}
	workflow, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	_, step, ok := strings.Cut(string(workflow), "      - name: Upload to draft release\n")
	if !ok {
		t.Fatal("release upload step is missing")
	}
	_, body, ok := strings.Cut(step, "        run: |\n")
	if !ok {
		t.Fatal("release upload script is missing")
	}
	var script strings.Builder
	for _, line := range strings.Split(body, "\n") {
		if line == "" {
			continue
		}
		line, ok = strings.CutPrefix(line, "          ")
		if !ok {
			break
		}
		script.WriteString(line + "\n")
	}

	for _, tt := range []struct {
		name      string
		state     string
		wantError bool
		wantWrite string
	}{
		{name: "published", state: "false", wantError: true},
		{name: "unknown state", state: "null", wantError: true},
		{name: "draft", state: "true", wantWrite: "release upload v0.9.4"},
		{name: "missing", state: "missing", wantWrite: "release create v0.9.4"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			calls := filepath.Join(dir, "writes")
			fakeGH := "#!/bin/sh\n" +
				"if [ \"$2\" = view ]; then\n" +
				"  [ \"$RELEASE_STATE\" != missing ] || exit 1\n" +
				"  printf '%s\\n' \"$RELEASE_STATE\"\n" +
				"else\n" +
				"  printf '%s\\n' \"$*\" >> \"$GH_WRITES\"\n" +
				"fi\n"
			if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(fakeGH), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("TAG", "v0.9.4")
			t.Setenv("RELEASE_STATE", tt.state)
			t.Setenv("GH_WRITES", calls)
			cmd := exec.Command(bash, "-e", "-o", "pipefail", "-c", script.String())
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			if (err != nil) != tt.wantError {
				t.Fatalf("upload error = %v; want error %v; output: %s", err, tt.wantError, output)
			}
			writes, err := os.ReadFile(calls)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if tt.wantWrite == "" && len(writes) != 0 {
				t.Fatalf("release was modified: %s", writes)
			}
			if tt.wantWrite != "" && !strings.HasPrefix(string(writes), tt.wantWrite+" ") {
				t.Fatalf("release writes = %q; want %q", writes, tt.wantWrite)
			}
			if tt.state == "missing" && !strings.Contains(string(writes), "--draft --verify-tag") {
				t.Fatalf("new release must be a draft for an existing tag: %s", writes)
			}
		})
	}
}
