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

	tea "charm.land/bubbletea/v2"
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

// press feeds keys to the model, e.g. press(m, "j", "enter", "esc").
func press(t *testing.T, m settingsModel, keys ...string) settingsModel {
	t.Helper()
	special := map[string]rune{"enter": tea.KeyEnter, "esc": tea.KeyEscape, "down": tea.KeyDown, "up": tea.KeyUp, "backspace": tea.KeyBackspace, "space": tea.KeySpace}
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch {
		case special[k] != 0:
			msg = tea.KeyPressMsg{Code: special[k]}
			if k == "space" {
				msg.Text = " "
			}
		case k == "ctrl+s":
			msg = tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
		default:
			// Typed text, one key per rune.
			for _, r := range k {
				next, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
				m = next.(settingsModel)
			}
			continue
		}
		next, _ := m.Update(msg)
		m = next.(settingsModel)
	}
	return m
}

func fieldIndex(t *testing.T, key string) int {
	t.Helper()
	for i, f := range SettingFields() {
		if f.Key == key {
			return i
		}
	}
	t.Fatalf("no setting %q", key)
	return 0
}

func TestSettingsTUINavigation(t *testing.T) {
	last := len(SettingFields()) - 1
	tests := []struct {
		keys []string
		want int
	}{
		{keys: []string{"j"}, want: 1},
		{keys: []string{"down", "down"}, want: 2},
		{keys: []string{"j", "j", "k"}, want: 1},
		{keys: []string{"up"}, want: 0},
		{keys: []string{"G"}, want: last},
		{keys: []string{"G", "j"}, want: last},
		{keys: []string{"G", "g"}, want: 0},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.keys, " "), func(t *testing.T) {
			z := newSettingsTestZVM(t)
			m := press(t, newSettingsModel(&z.Settings), tt.keys...)
			if m.cursor != tt.want {
				t.Fatalf("cursor = %d, want %d", m.cursor, tt.want)
			}
		})
	}
}

func TestSettingsTUIEditing(t *testing.T) {
	z := newSettingsTestZVM(t)
	m := newSettingsModel(&z.Settings)

	// Toggle a bool with space, then save with w.
	m.cursor = fieldIndex(t, "useColor")
	m = press(t, m, "space")
	if z.Settings.UseColor || !m.dirty() {
		t.Fatal("space did not toggle useColor")
	}
	if readSettings(t, z.Settings.Path()).UseColor != true {
		t.Fatal("toggle was saved before w")
	}
	m = press(t, m, "w")
	if readSettings(t, z.Settings.Path()).UseColor || m.dirty() {
		t.Fatal("w did not save")
	}

	// An invalid value is rejected and editing continues.
	m.cursor = fieldIndex(t, "versionMapUrl")
	m = press(t, m, "i")
	for range len(z.Settings.VersionMapUrl) {
		m = press(t, m, "backspace")
	}
	m = press(t, m, "not a url", "enter")
	if m.mode != modeEdit || !m.statusErr || !strings.Contains(m.status, "invalid value") {
		t.Fatalf("invalid value accepted: mode=%v status=%q", m.mode, m.status)
	}
	if z.Settings.VersionMapUrl != DefaultSettings.VersionMapUrl {
		t.Fatalf("invalid value applied: %q", z.Settings.VersionMapUrl)
	}

	// esc cancels the edit.
	m = press(t, m, "esc")
	if m.mode != modeBrowse || m.dirty() {
		t.Fatal("esc did not cancel")
	}

	// A valid value is applied, then reset to default with r.
	m = press(t, m, "i")
	for range len(z.Settings.VersionMapUrl) {
		m = press(t, m, "backspace")
	}
	m = press(t, m, "https://machengine.org/zig/index.json", "enter")
	if z.Settings.VersionMapUrl != MachVersionMapUrl {
		t.Fatalf("valid value not applied: %q (status %q)", z.Settings.VersionMapUrl, m.status)
	}
	m = press(t, m, "r")
	if z.Settings.VersionMapUrl != DefaultSettings.VersionMapUrl {
		t.Fatal("r did not reset")
	}
}

func TestSettingsTUIQuit(t *testing.T) {
	quits := func(m settingsModel, keys ...string) (settingsModel, bool) {
		for _, k := range keys {
			var msg tea.KeyPressMsg
			if k == "enter" {
				msg = tea.KeyPressMsg{Code: tea.KeyEnter}
			} else {
				msg = tea.KeyPressMsg{Code: rune(k[0]), Text: k}
			}
			// Running the returned commands would wait on the cursor blink
			// timer, so check the flag set alongside tea.Quit instead.
			next, _ := m.Update(msg)
			m = next.(settingsModel)
			if m.quitting {
				return m, true
			}
		}
		return m, false
	}

	tests := []struct {
		name     string
		dirty    bool
		keys     []string
		wantQuit bool
		wantSave bool
	}{
		{name: "q when clean", keys: []string{"q"}, wantQuit: true},
		{name: "q when dirty asks first", dirty: true, keys: []string{"q"}},
		{name: "q q when dirty", dirty: true, keys: []string{"q", "q"}, wantQuit: true},
		{name: ":q when dirty", dirty: true, keys: []string{":", "q", "enter"}},
		{name: ":q! when dirty", dirty: true, keys: []string{":", "q", "!", "enter"}, wantQuit: true},
		{name: ":wq when dirty", dirty: true, keys: []string{":", "w", "q", "enter"}, wantQuit: true, wantSave: true},
		{name: ":x when dirty", dirty: true, keys: []string{":", "x", "enter"}, wantQuit: true, wantSave: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := newSettingsTestZVM(t)
			m := newSettingsModel(&z.Settings)
			if tt.dirty {
				m.cursor = fieldIndex(t, "alwaysInstallZLS")
				m = press(t, m, "space")
			}
			_, quit := quits(m, tt.keys...)
			if quit != tt.wantQuit {
				t.Fatalf("quit = %v, want %v", quit, tt.wantQuit)
			}
			if saved := readSettings(t, z.Settings.Path()).AlwaysInstallZLS; saved != tt.wantSave {
				t.Fatalf("saved = %v, want %v", saved, tt.wantSave)
			}
		})
	}
}

func TestSettingsTUIShowsFileProblems(t *testing.T) {
	z := newSettingsTestZVM(t)
	if err := os.WriteFile(z.Settings.Path(), []byte(`{"useColor": "yes"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := z.loadSettings(); err != nil {
		t.Fatal(err)
	}
	z.Settings.UseColor = false // plain output for matching
	m := newSettingsModel(&z.Settings)
	view := m.View().Content
	if !strings.Contains(view, "1 invalid entry") || !strings.Contains(view, "✗ .useColor") && !strings.Contains(view, "expected a boolean") {
		t.Fatalf("view does not report the problem:\n%s", view)
	}

	// Fixing the field and saving clears the problem.
	m.cursor = fieldIndex(t, "useColor")
	m = press(t, m, "space", "w")
	if strings.Contains(m.View().Content, "invalid") {
		t.Fatalf("problem still shown after save:\n%s", m.View().Content)
	}
	if problems, _ := CheckSettingsFile(z.Settings.Path()); len(problems) != 0 {
		t.Fatalf("file still invalid after save: %v", problems)
	}
}
