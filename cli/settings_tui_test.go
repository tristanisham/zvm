//go:build !plan9

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

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
