// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseSettingPath(t *testing.T) {
	tests := []struct {
		path    string
		want    string
		wantErr bool
	}{
		{path: ".", want: ""},
		{path: ".useColor", want: "useColor"},
		{path: `."useColor"`, want: "useColor"},
		{path: `.["useColor"]`, want: "useColor"},
		{path: `."key with space"`, want: "key with space"},
		{path: "._private1", want: "_private1"},
		{path: "useColor", wantErr: true},
		{path: "", wantErr: true},
		{path: "..", wantErr: true},
		{path: ".useColor.nested", wantErr: true},
		{path: `.["useColor"`, wantErr: true},
		{path: `."useColor`, wantErr: true},
		{path: ".1abc", wantErr: true},
		{path: ".use-color", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := ParseSettingPath(tt.path)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidSettingPath) {
					t.Fatalf("ParseSettingPath(%q) = %q, %v; want ErrInvalidSettingPath", tt.path, got, err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("ParseSettingPath(%q) = %q, %v; want %q", tt.path, got, err, tt.want)
			}
		})
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		key     string
		raw     string
		want    any
		wantErr error
	}{
		{key: "useColor", raw: "false", want: false},
		{key: "useColor", raw: "true", want: true},
		{key: "useColor", raw: "yes", wantErr: ErrInvalidSetting},
		{key: "useColor", raw: `"true"`, wantErr: ErrInvalidSetting},
		{key: "mirrorListUrl", raw: "disabled", want: "disabled"},
		{key: "mirrorListUrl", raw: "https://example.com/mirrors.txt", want: "https://example.com/mirrors.txt"},
		{key: "mirrorListUrl", raw: `"https://example.com/m.txt"`, want: "https://example.com/m.txt"},
		{key: "mirrorListUrl", raw: "ftp://example.com/m.txt", wantErr: ErrInvalidSetting},
		{key: "mirrorListUrl", raw: "", wantErr: ErrInvalidSetting},
		{key: "versionMapUrl", raw: "disabled", wantErr: ErrInvalidSetting},
		{key: "versionMapUrl", raw: `"unterminated`, wantErr: ErrInvalidSetting},
		{key: "zlsVersionMapUrl", raw: "https:///nohost", wantErr: ErrInvalidSetting},
		{key: "minisignPubKey", raw: DefaultSettings.MinisignPubKey, want: DefaultSettings.MinisignPubKey},
		{key: "minisignPubKey", raw: "not-a-key", wantErr: ErrInvalidSetting},
	}

	for _, tt := range tests {
		t.Run(tt.key+"="+tt.raw, func(t *testing.T) {
			f, err := LookupSetting(tt.key)
			if err != nil {
				t.Fatal(err)
			}
			got, err := f.ParseValue(tt.raw)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ParseValue(%q) = %v, %v; want %v", tt.raw, got, err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("ParseValue(%q) = %v, %v; want %v", tt.raw, got, err, tt.want)
			}
		})
	}
}

func TestSettingFieldsCoverSettingsJSON(t *testing.T) {
	data, err := json.Marshal(DefaultSettings)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	fields := SettingFields()
	if len(fields) != len(m) {
		t.Fatalf("SettingFields has %d keys, settings.json has %d", len(fields), len(m))
	}
	for _, f := range fields {
		if _, ok := m[f.Key]; !ok {
			t.Errorf("SettingFields key %q is not written to settings.json", f.Key)
		}
		if f.Usage == "" {
			t.Errorf("setting %q has no usage text", f.Key)
		}
	}
}

func TestCheckSettingsFile(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		wantKeys []string // problem keys, "" for a whole-file problem
		wantText string
	}{
		{name: "valid", contents: mustJSON(t, DefaultSettings)},
		{name: "empty object", contents: `{}`},
		{name: "empty strings are defaults", contents: `{"mirrorListUrl": ""}`},
		{name: "malformed", contents: "{\n  \"useColor\": true,\n  \"x\"\n}", wantKeys: []string{""}, wantText: "line 4, column 1"},
		{name: "not an object", contents: `[1, 2]`, wantKeys: []string{""}},
		{name: "null object", contents: `null`, wantKeys: []string{""}, wantText: "JSON object"},
		{name: "null values", contents: `{"useColor": null, "versionMapUrl": null}`, wantKeys: []string{"versionMapUrl", "useColor"}},
		{name: "wrong type", contents: `{"useColor": "yes", "mirrorListUrl": 5}`, wantKeys: []string{"mirrorListUrl", "useColor"}},
		{name: "invalid url", contents: `{"versionMapUrl": "file:///etc/passwd"}`, wantKeys: []string{"versionMapUrl"}},
		{name: "invalid key", contents: `{"minisignPubKey": "abc"}`, wantKeys: []string{"minisignPubKey"}},
		{name: "unknown key", contents: `{"useColour": true}`, wantKeys: []string{"useColour"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if err := os.WriteFile(path, []byte(tt.contents), 0o644); err != nil {
				t.Fatal(err)
			}
			problems, err := CheckSettingsFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var keys []string
			var text strings.Builder
			for _, p := range problems {
				keys = append(keys, p.Key)
				text.WriteString(p.String() + "\n")
			}
			if strings.Join(keys, ",") != strings.Join(tt.wantKeys, ",") {
				t.Fatalf("problem keys = %q, want %q\n%s", keys, tt.wantKeys, text.String())
			}
			if !strings.Contains(text.String(), tt.wantText) {
				t.Fatalf("problems %q do not mention %q", text.String(), tt.wantText)
			}
		})
	}
}

// A broken settings.json must not leave ZVM running with empty settings.
func TestLoadSettingsUsesDefaultsForInvalidEntries(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "settings.json")
	contents := `{"useColor": "yes", "versionMapUrl": "nope", "mirrorListUrl": "disabled", "alwaysInstallZLS": true}`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	z := &ZVM{baseDir: base}
	z.Settings.path = path
	if err := z.loadSettings(); err != nil {
		t.Fatal(err)
	}

	s := z.Settings
	if s.UseColor != DefaultSettings.UseColor || s.VersionMapUrl != DefaultSettings.VersionMapUrl {
		t.Errorf("invalid entries not defaulted: useColor=%v versionMapUrl=%q", s.UseColor, s.VersionMapUrl)
	}
	if s.MirrorListUrl != "disabled" || !s.AlwaysInstallZLS {
		t.Errorf("valid entries not kept: mirrorListUrl=%q alwaysInstallZLS=%v", s.MirrorListUrl, s.AlwaysInstallZLS)
	}
	if s.MinisignPubKey != DefaultSettings.MinisignPubKey {
		t.Errorf("missing entry not defaulted: minisignPubKey=%q", s.MinisignPubKey)
	}
	if len(s.Problems()) != 2 {
		t.Errorf("Problems() = %v, want 2", s.Problems())
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != contents {
		t.Errorf("loading rewrote the invalid file:\n%s", after)
	}
}

func TestLoadSettingsMalformedFileUsesDefaults(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "settings.json")
	if err := os.WriteFile(path, []byte(`{"useColor": `), 0o644); err != nil {
		t.Fatal(err)
	}
	z := &ZVM{baseDir: base}
	z.Settings.path = path
	if err := z.loadSettings(); err != nil {
		t.Fatal(err)
	}
	got := z.Settings
	got.path, got.problems = "", nil
	if !reflect.DeepEqual(got, DefaultSettings) {
		t.Errorf("settings = %+v, want defaults", got)
	}
	if len(z.Settings.Problems()) != 1 || z.Settings.Path() != path {
		t.Errorf("problems = %v, path = %q", z.Settings.Problems(), z.Settings.Path())
	}
}

func TestSettingsCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		raw     bool
		want    string
		wantErr error
		check   func(t *testing.T, s Settings)
	}{
		{name: "get bool", args: []string{".useColor"}, want: "true\n"},
		{name: "get string", args: []string{".mirrorListUrl"}, want: `"` + DefaultSettings.MirrorListUrl + "\"\n"},
		{name: "get raw string", args: []string{".mirrorListUrl"}, raw: true, want: DefaultSettings.MirrorListUrl + "\n"},
		{name: "get quoted path", args: []string{`.["useColor"]`}, want: "true\n"},
		{name: "get whole file", args: []string{"."}, want: `"useColor": true`},
		{
			name: "set bool", args: []string{".useColor", "false"}, want: ".useColor = false\n",
			check: func(t *testing.T, s Settings) {
				if s.UseColor {
					t.Error("useColor not saved")
				}
			},
		},
		{
			name: "set string", args: []string{".mirrorListUrl", "disabled"}, want: `.mirrorListUrl = "disabled"`,
			check: func(t *testing.T, s Settings) {
				if s.MirrorListUrl != "disabled" {
					t.Errorf("mirrorListUrl = %q", s.MirrorListUrl)
				}
			},
		},
		{name: "set invalid", args: []string{".versionMapUrl", "not a url"}, wantErr: ErrInvalidSetting, check: unchanged},
		{name: "set unknown", args: []string{".useColour", "true"}, wantErr: ErrUnknownSetting, check: unchanged},
		{name: "set nested", args: []string{".useColor.x", "true"}, wantErr: ErrInvalidSettingPath, check: unchanged},
		{name: "replace whole file", args: []string{".", "{}"}, wantErr: ErrInvalidSettingPath, check: unchanged},
		{name: "too many args", args: []string{".useColor", "true", "x"}, wantErr: ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := newSettingsTestZVM(t)
			var out bytes.Buffer
			err := z.SettingsCommand(tt.args, false, tt.raw, &out)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("output %q does not contain %q", out.String(), tt.want)
			}
			if tt.check != nil {
				tt.check(t, readSettings(t, z.Settings.Path()))
			}
		})
	}
}

func TestSettingsCommandCheck(t *testing.T) {
	z := newSettingsTestZVM(t)
	var out bytes.Buffer
	if err := z.SettingsCommand(nil, true, false, &out); err != nil || !strings.Contains(out.String(), "is valid") {
		t.Fatalf("valid file: err=%v out=%q", err, out.String())
	}

	if err := os.WriteFile(z.Settings.Path(), []byte(`{"useColor": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	err := z.SettingsCommand(nil, true, false, &out)
	if !errors.Is(err, ErrSettingsProblems) || !strings.Contains(out.String(), ".useColor") {
		t.Fatalf("invalid file: err=%v out=%q", err, out.String())
	}
}

func TestSettingsCommandOnFirstRun(t *testing.T) {
	base := filepath.Join(t.TempDir(), "new-install")
	t.Setenv("ZVM_PATH", base)
	z := Initialize()
	var out bytes.Buffer
	if err := z.SettingsCommand([]string{".useColor", "false"}, false, false, &out); err != nil {
		t.Fatal(err)
	}
	if path := filepath.Join(base, "settings.json"); z.Settings.Path() != path || readSettings(t, path).UseColor {
		t.Fatalf("first-run settings were not saved to %s", path)
	}
}

func TestLoadSettingsRejectsNullWithoutRewriting(t *testing.T) {
	for _, contents := range []string{`null`, `{"useColor":null}`, `{"versionMapUrl":null}`} {
		t.Run(contents, func(t *testing.T) {
			base := t.TempDir()
			path := filepath.Join(base, "settings.json")
			if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			z := &ZVM{baseDir: base}
			z.Settings.path = path
			if err := z.loadSettings(); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := z.SettingsCommand(nil, true, false, &out); !errors.Is(err, ErrSettingsProblems) {
				t.Fatalf("settings --check = %v, want ErrSettingsProblems", err)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != contents {
				t.Fatalf("invalid settings were changed: %q, %v", after, err)
			}
		})
	}
}

func unchanged(t *testing.T, s Settings) {
	t.Helper()
	s.path = ""
	if !reflect.DeepEqual(s, DefaultSettings) {
		t.Errorf("settings changed after a rejected command: %+v", s)
	}
}

func newSettingsTestZVM(t *testing.T) *ZVM {
	t.Helper()
	z := &ZVM{baseDir: t.TempDir(), Settings: DefaultSettings}
	z.Settings.path = filepath.Join(z.baseDir, "settings.json")
	if err := z.Settings.Save(); err != nil {
		t.Fatal(err)
	}
	return z
}

func readSettings(t *testing.T, path string) Settings {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
