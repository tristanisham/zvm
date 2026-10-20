// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/x/term"
)

// SettingsCommand implements `zvm settings [.key [value]]`.
//
// With no path it opens the editor when attached to a terminal and prints
// settings.json otherwise. A path prints that value; a path and a value
// validate the value and save it.
func (z *ZVM) SettingsCommand(args []string, check, raw bool, out io.Writer) error {
	if check {
		return z.checkSettings(out)
	}

	switch len(args) {
	case 0:
		if term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd()) {
			return z.RunSettingsTUI(os.Stdin, os.Stdout)
		}
		return printSettingJSON(out, z.Settings, false)
	case 1, 2:
	default:
		return fmt.Errorf("%w: expected at most a path and a value, got %d arguments", ErrInvalidInput, len(args))
	}

	key, err := ParseSettingPath(args[0])
	if err != nil {
		return err
	}
	if key == "" {
		if len(args) == 2 {
			return fmt.Errorf("%w: cannot replace the whole file; set one key at a time, e.g. .useColor false", ErrInvalidSettingPath)
		}
		return printSettingJSON(out, z.Settings, false)
	}

	if len(args) == 2 {
		if err := z.Settings.Set(key, args[1]); err != nil {
			return err
		}
	}

	v, err := z.Settings.Get(key)
	if err != nil {
		return err
	}
	if len(args) == 2 {
		fmt.Fprintf(out, ".%s = ", key)
	}
	return printSettingJSON(out, v, raw)
}

func (z *ZVM) checkSettings(out io.Writer) error {
	problems, err := CheckSettingsFile(z.Settings.Path())
	if err != nil {
		return err
	}
	if len(problems) == 0 {
		fmt.Fprintf(out, "%s is valid\n", z.Settings.Path())
		return nil
	}
	for _, p := range problems {
		fmt.Fprintln(out, p.String())
	}
	return fmt.Errorf("%w: %d invalid entr%s in %s", ErrSettingsProblems, len(problems), plural(len(problems), "y", "ies"), z.Settings.Path())
}

func printSettingJSON(out io.Writer, v any, raw bool) error {
	if s, ok := v.(string); ok && raw {
		_, err := fmt.Fprintln(out, s)
		return err
	}
	data, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(data))
	return err
}
