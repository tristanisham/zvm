// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/tristanisham/clr"
)

// aliasFileName holds user-defined names for installed Zig versions, as a
// JSON object of name to version.
//
// A file rather than a database: the data is a handful of short strings that
// are read once per command and written only when the user edits them, and an
// embedded SQL engine costs every platform Go can target. modernc.org/sqlite
// has no support for solaris or netbsd/arm64, which zvm releases for.
const aliasFileName = "aliases.json"

// Alias is a user-defined name for an installed Zig version.
type Alias struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Alias creates or updates key when val is provided, or prints the existing
// alias when val is nil. Alias values must identify an installed Zig version.
func (z *ZVM) Alias(ctx context.Context, key string, val *string) error {
	if key == "" {
		return ErrMissingArgument
	}

	aliases, err := z.readAliases()
	if err != nil {
		return err
	}

	if val == nil {
		value, ok := aliases[key]
		if !ok {
			return ErrInvalidAlias
		}

		z.PrintAliases([]Alias{{Key: key, Value: value}})
		return nil
	}

	normalizedVal := strings.TrimPrefix(*val, "v")
	installedVersions, err := z.GetInstalledVersions()
	if err != nil {
		return err
	}

	resolvedVal, err := resolveVersionShorthand(normalizedVal, installedVersions)
	if err != nil {
		return err
	}
	if !slices.Contains(installedVersions, resolvedVal) {
		return ErrInvalidAliasValue
	}

	aliases[key] = resolvedVal
	if err := z.writeAliases(aliases); err != nil {
		return fmt.Errorf("%w: %w", ErrFailedAliasSave, err)
	}

	return nil
}

// ResolveAlias looks up key. A missing key is reported by returning ok=false.
func (z *ZVM) ResolveAlias(ctx context.Context, key string) (value string, ok bool, err error) {
	if key == "" {
		return "", false, nil
	}

	aliases, err := z.readAliases()
	if err != nil {
		return "", false, err
	}

	value, ok = aliases[key]
	return value, ok, nil
}

// DeleteAlias deletes key. Deleting a key that does not exist is successful.
func (z *ZVM) DeleteAlias(ctx context.Context, key string) error {
	if key == "" {
		return ErrMissingArgument
	}

	aliases, err := z.readAliases()
	if err != nil {
		return err
	}
	if _, ok := aliases[key]; !ok {
		return nil
	}

	delete(aliases, key)
	if err := z.writeAliases(aliases); err != nil {
		return fmt.Errorf("%w: %w", ErrFailedAliasClear, err)
	}
	return nil
}

// ClearAliases deletes all aliases.
func (z *ZVM) ClearAliases(ctx context.Context) error {
	if err := z.writeAliases(map[string]string{}); err != nil {
		return fmt.Errorf("%w: %w", ErrFailedAliasClear, err)
	}
	return nil
}

// ListAliases returns all aliases ordered by key.
func (z *ZVM) ListAliases(ctx context.Context) ([]Alias, error) {
	aliases, err := z.readAliases()
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(aliases))
	for key := range aliases {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]Alias, 0, len(keys))
	for _, key := range keys {
		out = append(out, Alias{Key: key, Value: aliases[key]})
	}
	return out, nil
}

// aliasPath is where aliases are stored for this ZVM instance.
func (z *ZVM) aliasPath() string {
	return filepath.Join(z.baseDir, aliasFileName)
}

// readAliases loads the alias file. Having no aliases yet is not an error, so
// a missing file reads as an empty set.
func (z *ZVM) readAliases() (map[string]string, error) {
	contents, err := os.ReadFile(z.aliasPath())
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadDatabase, err)
	}

	// An empty file is what a previously interrupted write leaves behind;
	// treat it the same as no aliases rather than as corruption.
	if len(strings.TrimSpace(string(contents))) == 0 {
		return map[string]string{}, nil
	}

	aliases := make(map[string]string)
	if err := json.Unmarshal(contents, &aliases); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrBadDatabase, z.aliasPath(), err)
	}

	return aliases, nil
}

// writeAliases replaces the alias file. The write goes to a temporary file in
// the same directory and is renamed into place, so an interrupted write cannot
// leave a half-written file where the aliases used to be.
func (z *ZVM) writeAliases(aliases map[string]string) error {
	contents, err := json.MarshalIndent(aliases, "", "    ")
	if err != nil {
		return err
	}
	contents = append(contents, '\n')

	temp, err := os.CreateTemp(z.baseDir, aliasFileName+".*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())

	if _, err := temp.Write(contents); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temp.Name(), 0644); err != nil {
		return err
	}

	return os.Rename(temp.Name(), z.aliasPath())
}

// PrintAliases prints aliases one per line.
func (z *ZVM) PrintAliases(aliases []Alias) {
	for _, alias := range aliases {
		if z.Settings.UseColor {
			fmt.Printf("%s %s\n", clr.Blue(alias.Key), clr.White(alias.Value))
		} else {
			fmt.Printf("%s %s\n", alias.Key, alias.Value)
		}
	}
}
