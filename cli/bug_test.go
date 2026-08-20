// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanisham/zvm/cli/meta"
)

func newBugZVM(t *testing.T) *ZVM {
	t.Helper()

	base := t.TempDir()
	for _, version := range []string{"0.13.0", "master"} {
		if err := os.Mkdir(filepath.Join(base, version), 0755); err != nil {
			t.Fatalf("create installed version %s: %v", version, err)
		}
	}
	t.Setenv("ZVM_PATH", base)

	return Initialize()
}

func TestBugReportBodyHasTemplateSections(t *testing.T) {
	z := newBugZVM(t)
	body := z.BugReportBody()

	for _, section := range []string{
		"### What version of ZVM are you using (`zvm version`)?",
		"### Does this issue reproduce with the latest release?",
		"### What operating system and processor architecture are you using?",
		"### What did you do?",
		"### What did you see happen?",
		"### What did you expect to see?",
	} {
		if !strings.Contains(body, section) {
			t.Errorf("bug report body is missing section %q", section)
		}
	}

	if !strings.Contains(body, "<details><summary><code>zvm bug</code> Output</summary>") {
		t.Error("bug report body does not wrap the environment dump in a <details> block")
	}
}

func TestBugReportBodyReportsEnvironment(t *testing.T) {
	z := newBugZVM(t)
	body := z.BugReportBody()

	for _, want := range []string{
		"ZVM_VERSION='" + meta.VERSION + "'",
		"GOOS='",
		"GOARCH='",
		"GOVERSION='",
		"ZVM_PATH='" + redactPath(z.baseDir) + "'",
		"VERSION_MAP_URL='" + z.Settings.VersionMapUrl + "'",
		"INSTALLED_ZIG='0.13.0 master'",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("bug report body does not report %q", want)
		}
	}
}

func TestBugReportBodyRedactsUnsetEnvironmentVariables(t *testing.T) {
	z := newBugZVM(t)
	t.Setenv("ZVM_DEBUG", "1")
	os.Unsetenv("ZVM_SKIP_TLS_VERIFY")

	body := z.BugReportBody()

	if !strings.Contains(body, "ZVM_DEBUG='1'") {
		t.Error("bug report body does not report a set ZVM_DEBUG")
	}
	if !strings.Contains(body, "ZVM_SKIP_TLS_VERIFY=''") {
		t.Error("bug report body does not report an unset ZVM_SKIP_TLS_VERIFY as empty")
	}
}

func TestBugReportURLRoundTrips(t *testing.T) {
	body := "### What did you do?\n\nRan `zvm i master` & waited.\n"

	got := BugReportURL(body)

	const prefix = "https://github.com/tristanisham/zvm/issues/new?body="
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("BugReportURL() = %q, want prefix %q", got, prefix)
	}

	decoded, err := url.QueryUnescape(strings.TrimPrefix(got, prefix))
	if err != nil {
		t.Fatalf("decoding query: %v", err)
	}
	if decoded != body {
		t.Errorf("decoded body = %q, want %q", decoded, body)
	}

	if strings.ContainsAny(strings.TrimPrefix(got, prefix), " \n") {
		t.Error("BugReportURL() left unescaped whitespace in the query")
	}
}

func TestBugReportRedactsHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir reads this on Windows.

	base := filepath.Join(home, ".zvm")
	if err := os.Mkdir(base, 0755); err != nil {
		t.Fatalf("create base dir: %v", err)
	}
	t.Setenv("ZVM_PATH", base)
	t.Setenv("ZVM_INSTALL", filepath.Join(base, "self"))

	body := Initialize().BugReportBody()

	if strings.Contains(body, home) {
		t.Errorf("bug report body leaks the home directory %q", home)
	}
	for _, want := range []string{
		"ZVM_PATH='" + filepath.Join("~", ".zvm") + "'",
		"ZVM_INSTALL='" + filepath.Join("~", ".zvm", "self") + "'",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("bug report body does not contain %q", want)
		}
	}
}

func TestRedactPathLeavesPathsOutsideHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	outside := filepath.Join(t.TempDir(), "zvm")
	if got := redactPath(outside); got != outside {
		t.Errorf("redactPath(%q) = %q, want it unchanged", outside, got)
	}

	// A sibling directory that merely shares the home prefix is not inside it.
	sibling := home + "-backup"
	if got := redactPath(sibling); got != sibling {
		t.Errorf("redactPath(%q) = %q, want it unchanged", sibling, got)
	}

	if got := redactPath(""); got != "" {
		t.Errorf("redactPath(\"\") = %q, want empty", got)
	}
}

func TestRedactSettingKeepsPublicValues(t *testing.T) {
	for _, value := range []string{
		"",
		"disabled",
		DefaultSettings.MirrorListUrl,
		DefaultSettings.VersionMapUrl,
		DefaultSettings.ZlsVMU,
		DefaultSettings.MinisignPubKey,
		MachVersionMapUrl,
	} {
		if got := redactSetting(value); got != value {
			t.Errorf("redactSetting(%q) = %q, want it reported verbatim", value, got)
		}
	}
}

func TestRedactSettingDigestsCustomValues(t *testing.T) {
	const private = "https://zig.internal.example.com/index.json"

	got := redactSetting(private)

	if strings.Contains(got, "internal.example.com") {
		t.Errorf("redactSetting(%q) = %q, want the host withheld", private, got)
	}
	if !strings.HasPrefix(got, "custom sha256:") {
		t.Errorf("redactSetting(%q) = %q, want a custom digest", private, got)
	}
	// The digest is a correlation tag: the same value must always tag the same.
	if again := redactSetting(private); again != got {
		t.Errorf("redactSetting is not stable: %q then %q", got, again)
	}
	if other := redactSetting(private + "/v2"); other == got {
		t.Errorf("distinct values share the tag %q", got)
	}
}

func TestBugReportRedactsCustomVersionMap(t *testing.T) {
	z := newBugZVM(t)
	z.Settings.VersionMapUrl = "https://zig.corp.example.net/index.json"

	body := z.BugReportBody()

	if strings.Contains(body, "corp.example.net") {
		t.Error("bug report body leaks a custom version-map host")
	}
	if !strings.Contains(body, "VERSION_MAP_URL='custom sha256:") {
		t.Error("bug report body does not tag the custom version map")
	}
}
