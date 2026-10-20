// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type settingsMode int

const (
	modeBrowse settingsMode = iota
	modeEdit
	modeCommand
)

// settingsModel is the Bubble Tea model behind `zvm settings`.
type settingsModel struct {
	settings *Settings // edited in place; written by save
	saved    Settings  // what is on disk, to detect unsaved changes
	fields   []SettingField
	problems map[string]string // key -> problem found in settings.json
	fileErr  string            // problem with the file as a whole

	cursor     int
	mode       settingsMode
	input      textinput.Model
	status     string
	statusErr  bool
	confirmBye bool // q pressed once with unsaved changes
	quitting   bool
	width      int // terminal width; 0 until the first WindowSizeMsg

	styles settingsStyles
}

type settingsStyles struct {
	title, key, value, dim, cursor, err, ok, warn lipgloss.Style
}

func newSettingsStyles(color bool) settingsStyles {
	if !color {
		plain := lipgloss.NewStyle()
		return settingsStyles{plain, plain, plain, plain, plain.Bold(true), plain, plain, plain}
	}
	return settingsStyles{
		title:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F7A41D")),
		key:    lipgloss.NewStyle().Bold(true),
		value:  lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		dim:    lipgloss.NewStyle().Faint(true),
		cursor: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F7A41D")),
		err:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		ok:     lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		warn:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
	}
}

func newSettingsModel(s *Settings) settingsModel {
	m := settingsModel{
		settings: s,
		saved:    *s,
		fields:   SettingFields(),
		problems: map[string]string{},
		input:    textinput.New(),
		styles:   newSettingsStyles(s.UseColor),
	}
	m.input.Prompt = ""
	for _, p := range s.Problems() {
		if p.Key == "" {
			m.fileErr = p.Err.Error()
		} else {
			m.problems[p.Key] = p.Err.Error()
		}
	}
	return m
}

func (m settingsModel) Init() tea.Cmd { return nil }

func (m settingsModel) current() SettingField { return m.fields[m.cursor] }

func (m settingsModel) value(f SettingField) any {
	v, _ := m.settings.Get(f.Key)
	return v
}

func (m settingsModel) dirty() bool {
	a, b := reflect.ValueOf(*m.settings), reflect.ValueOf(m.saved)
	for _, f := range m.fields {
		if a.Field(f.index).Interface() != b.Field(f.index).Interface() {
			return true
		}
	}
	return false
}

func (m *settingsModel) setStatus(msg string, isErr bool) {
	m.status, m.statusErr = msg, isErr
}

func (m settingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		// Leave room for the pointer, the key column and a margin.
		m.input.SetWidth(max(size.Width-m.keyWidth()-6, 10))
		return m, nil
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		if m.mode != modeBrowse {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch m.mode {
	case modeEdit:
		return m.updateEdit(key)
	case modeCommand:
		return m.updateCommand(key)
	}
	return m.updateBrowse(key)
}

func (m settingsModel) updateBrowse(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := key.String()
	if k != "q" && k != "esc" {
		m.confirmBye = false
	}

	switch k {
	case "j", "down", "ctrl+n", "tab":
		m.cursor = min(m.cursor+1, len(m.fields)-1)
		m.setStatus("", false)
	case "k", "up", "ctrl+p", "shift+tab":
		m.cursor = max(m.cursor-1, 0)
		m.setStatus("", false)
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(m.fields) - 1
	case "enter", "i", "l", "right", "space", " ":
		f := m.current()
		if f.Kind == reflect.Bool {
			b, _ := m.value(f).(bool)
			m.settings.setField(f, !b)
			delete(m.problems, f.Key)
			m.setStatus(fmt.Sprintf(".%s = %t (unsaved)", f.Key, !b), false)
			return m, nil
		}
		m.mode = modeEdit
		m.input.SetValue(fmt.Sprint(m.value(f)))
		m.input.CursorEnd()
		m.setStatus("enter to apply, esc to cancel", false)
		return m, m.input.Focus()
	case "r":
		f := m.current()
		def := reflect.ValueOf(DefaultSettings).Field(f.index).Interface()
		m.settings.setField(f, def)
		delete(m.problems, f.Key)
		m.setStatus(fmt.Sprintf(".%s reset to its default (unsaved)", f.Key), false)
	case "w", "ctrl+s":
		return m.save(false)
	case ":":
		m.mode = modeCommand
		m.input.SetValue("")
		m.setStatus("", false)
		return m, m.input.Focus()
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "q", "esc":
		if m.dirty() && !m.confirmBye {
			m.confirmBye = true
			m.setStatus("unsaved changes: w to save, q again to quit without saving", true)
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m settingsModel) updateEdit(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.mode = modeBrowse
		m.input.Blur()
		m.setStatus("edit cancelled", false)
		return m, nil
	case "enter":
		f := m.current()
		v, err := f.ParseValue(m.input.Value())
		if err != nil {
			// Stay in edit mode so the value can be corrected.
			m.setStatus(err.Error(), true)
			return m, nil
		}
		m.settings.setField(f, v)
		delete(m.problems, f.Key)
		m.mode = modeBrowse
		m.input.Blur()
		m.setStatus(fmt.Sprintf(".%s updated (unsaved)", f.Key), false)
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)
	return m, cmd
}

// updateCommand handles vim-style :w, :q, :wq, :x and :q!.
func (m settingsModel) updateCommand(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.mode = modeBrowse
		m.input.Blur()
		return m, nil
	case "enter":
		cmd := strings.TrimSpace(m.input.Value())
		m.mode = modeBrowse
		m.input.Blur()
		switch cmd {
		case "w":
			return m.save(false)
		case "wq", "x":
			return m.save(true)
		case "q":
			if m.dirty() {
				m.setStatus("unsaved changes: :w to save, :q! to quit without saving", true)
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case "q!":
			m.quitting = true
			return m, tea.Quit
		}
		m.setStatus(fmt.Sprintf("unknown command :%s (use :w, :q, :wq or :q!)", cmd), true)
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)
	return m, cmd
}

func (m settingsModel) save(quit bool) (tea.Model, tea.Cmd) {
	if err := m.settings.Save(); err != nil {
		m.setStatus("save failed: "+err.Error(), true)
		return m, nil
	}
	m.saved = *m.settings
	m.problems = map[string]string{}
	m.fileErr = ""
	m.confirmBye = false
	m.setStatus("saved "+m.settings.Path(), false)
	if quit {
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m settingsModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	st := m.styles
	var b strings.Builder

	b.WriteString(st.title.Render("ZVM settings"))
	b.WriteString(st.dim.Render("  " + m.settings.Path()))
	b.WriteString("\n")
	if m.fileErr != "" {
		b.WriteString(st.err.Render("✗ " + m.fileErr))
		b.WriteString("\n")
		b.WriteString(st.warn.Render("  Saving rewrites the file with the values shown below."))
		b.WriteString("\n")
	} else if len(m.problems) > 0 {
		b.WriteString(st.warn.Render(fmt.Sprintf("%d invalid entr%s in settings.json; defaults are shown for them. Saving replaces them.", len(m.problems), plural(len(m.problems), "y", "ies"))))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	width := m.keyWidth()

	for i, f := range m.fields {
		pointer := "  "
		keyStyle := st.key
		if i == m.cursor {
			pointer = st.cursor.Render("> ")
			keyStyle = st.cursor
		}
		b.WriteString(pointer)
		b.WriteString(keyStyle.Render(fmt.Sprintf("%-*s", width, "."+f.Key)))
		b.WriteString("  ")

		switch {
		case i == m.cursor && m.mode == modeEdit:
			b.WriteString(m.input.View())
		default:
			b.WriteString(st.value.Render(formatSettingValue(m.value(f))))
		}

		saved := reflect.ValueOf(m.saved).Field(f.index).Interface()
		if m.value(f) != saved {
			b.WriteString(st.warn.Render(" *"))
		}
		b.WriteString("\n")

		if p, bad := m.problems[f.Key]; bad {
			b.WriteString(st.err.Render(fmt.Sprintf("    ✗ %s", p)))
			b.WriteString("\n")
		} else if i == m.cursor && f.Usage != "" {
			b.WriteString(st.dim.Render("    " + f.Usage))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	switch {
	case m.mode == modeCommand:
		b.WriteString(":" + m.input.View())
	case m.status != "":
		if m.statusErr {
			b.WriteString(st.err.Render(m.status))
		} else {
			b.WriteString(st.ok.Render(m.status))
		}
	}
	b.WriteString("\n")

	var help string
	switch m.mode {
	case modeEdit:
		help = "enter apply • esc cancel"
	case modeCommand:
		help = ":w save • :q quit • :wq save and quit • :q! quit without saving"
	default:
		help = "j/k ↑/↓ move • g/G top/bottom • enter/i edit • space toggle • r reset • w save • : command • q quit"
	}
	b.WriteString(st.dim.Render(help))

	// Lines wider than the terminal would wrap; cut them instead.
	content := b.String()
	if m.width > 0 {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lines[i] = ansi.Truncate(line, m.width, "…")
		}
		content = strings.Join(lines, "\n")
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// keyWidth is the width of the key column, including the leading dot.
func (m settingsModel) keyWidth() int {
	width := 0
	for _, f := range m.fields {
		width = max(width, len(f.Key)+1)
	}
	return width
}

func formatSettingValue(v any) string {
	if s, ok := v.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprint(v)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// RunSettingsTUI opens the interactive settings editor.
func (z *ZVM) RunSettingsTUI(in io.Reader, out io.Writer) error {
	p := tea.NewProgram(newSettingsModel(&z.Settings), tea.WithInput(in), tea.WithOutput(out))
	final, err := p.Run()
	if err != nil {
		return err
	}
	if m, ok := final.(settingsModel); ok && m.dirty() {
		fmt.Fprintln(out, "Quit without saving; settings.json was not changed.")
	}
	return nil
}
