// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/jedisct1/go-minisign"
)

// SettingField describes one key of settings.json.
type SettingField struct {
	Key   string       // JSON key, e.g. "mirrorListUrl"
	Kind  reflect.Kind // reflect.String or reflect.Bool
	Usage string       // one-line description shown in `zvm settings`
	index int          // field index in Settings
}

// SettingProblem is one invalid entry found in settings.json. Key is empty
// when the problem concerns the whole file (e.g. malformed JSON).
type SettingProblem struct {
	Key string
	Err error
}

func (p SettingProblem) String() string {
	if p.Key == "" {
		return p.Err.Error()
	}
	return fmt.Sprintf(".%s: %v", p.Key, p.Err)
}

var settingUsage = map[string]string{
	"mirrorListUrl":      `community mirror list URL, or "disabled" to download from ziglang.org`,
	"minisignPubKey":     "public key that official Zig downloads are verified against",
	"versionMapUrl":      "Zig version map (index.json) URL",
	"zlsVersionMapUrl":   "ZLS release API URL",
	"useColor":           "colored terminal output",
	"alwaysForceInstall": "reinstall versions that are already installed",
	"alwaysInstallZLS":   "install ZLS together with Zig",
}

var settingValidators = map[string]func(string) error{
	"mirrorListUrl": func(v string) error {
		if v == "disabled" {
			return nil
		}
		return checkWebURL(v)
	},
	"minisignPubKey": func(v string) error {
		_, err := minisign.NewPublicKey(v)
		return err
	},
	"versionMapUrl":    checkWebURL,
	"zlsVersionMapUrl": checkWebURL,
}

// checkWebURL is isValidWebURL with a clearer message for values that are
// not URLs at all.
func checkWebURL(v string) error {
	if !strings.Contains(v, "://") {
		return fmt.Errorf("%q is not a URL (expected http:// or https://)", v)
	}
	return isValidWebURL(v)
}

// SettingFields lists the keys of settings.json in file order.
func SettingFields() []SettingField {
	t := reflect.TypeFor[Settings]()
	fields := make([]SettingField, 0, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		key, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if !f.IsExported() || key == "" || key == "-" {
			continue
		}
		fields = append(fields, SettingField{Key: key, Kind: f.Type.Kind(), Usage: settingUsage[key], index: i})
	}
	return fields
}

// LookupSetting returns the field for key.
func LookupSetting(key string) (SettingField, error) {
	fields := SettingFields()
	for _, f := range fields {
		if f.Key == key {
			return f, nil
		}
	}

	keys := make([]string, len(fields))
	for i, f := range fields {
		keys[i] = "." + f.Key
	}
	return SettingField{}, fmt.Errorf("%w %q (valid keys: %s)", ErrUnknownSetting, key, strings.Join(keys, ", "))
}

// ParseSettingPath parses a jq-style path such as `.useColor`, `."useColor"`
// or `.["useColor"]` and returns the key. The empty path `.` returns "".
// settings.json is a flat object, so nested paths are rejected.
func ParseSettingPath(path string) (string, error) {
	if path == "." {
		return "", nil
	}
	if !strings.HasPrefix(path, ".") {
		return "", fmt.Errorf("%w %q: paths start with '.', e.g. .useColor", ErrInvalidSettingPath, path)
	}

	var segments []string
	rest := path
	for rest != "" {
		var seg string
		var err error
		switch {
		case strings.HasPrefix(rest, `.["`):
			seg, rest, err = cutQuoted(rest[2:])
			if err == nil {
				if !strings.HasPrefix(rest, "]") {
					err = errors.New(`missing "]"`)
				} else {
					rest = rest[1:]
				}
			}
		case strings.HasPrefix(rest, `."`):
			seg, rest, err = cutQuoted(rest[1:])
		case strings.HasPrefix(rest, "."):
			end := strings.IndexAny(rest[1:], `.["`)
			if end < 0 {
				end = len(rest) - 1
			}
			seg, rest = rest[1:end+1], rest[end+1:]
			if !isIdentifier(seg) {
				err = fmt.Errorf("%q is not a plain key; quote it as .%q", seg, seg)
			}
		default:
			err = fmt.Errorf("unexpected %q", rest)
		}
		if err != nil {
			return "", fmt.Errorf("%w %q: %w", ErrInvalidSettingPath, path, err)
		}
		segments = append(segments, seg)
	}

	if len(segments) > 1 {
		return "", fmt.Errorf("%w %q: .%s is not an object, settings.json has no nested keys", ErrInvalidSettingPath, path, segments[0])
	}
	return segments[0], nil
}

// cutQuoted reads a JSON string literal at the start of s.
func cutQuoted(s string) (string, string, error) {
	dec := json.NewDecoder(strings.NewReader(s))
	var v string
	if err := dec.Decode(&v); err != nil {
		return "", "", fmt.Errorf("bad quoted key: %w", err)
	}
	return v, s[dec.InputOffset():], nil
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r != '_' && !('a' <= r && r <= 'z') && !('A' <= r && r <= 'Z') && (i == 0 || !('0' <= r && r <= '9')) {
			return false
		}
	}
	return true
}

// Get returns the current value of key.
func (s *Settings) Get(key string) (any, error) {
	f, err := LookupSetting(key)
	if err != nil {
		return nil, err
	}
	return reflect.ValueOf(s).Elem().Field(f.index).Interface(), nil
}

// ParseValue converts raw into a value for f and validates it. Booleans take
// true or false; strings may be given bare or as a JSON string literal.
func (f SettingField) ParseValue(raw string) (any, error) {
	v, err := f.parseValue(raw)
	if err != nil {
		return nil, fmt.Errorf(".%s: %w", f.Key, err)
	}
	return v, nil
}

func (f SettingField) parseValue(raw string) (any, error) {
	var v any
	switch f.Kind {
	case reflect.Bool:
		b, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("%w: %q is not true or false", ErrInvalidSetting, raw)
		}
		v = b
	case reflect.String:
		str := raw
		if strings.HasPrefix(strings.TrimSpace(raw), `"`) {
			if err := json.Unmarshal([]byte(raw), &str); err != nil {
				return nil, fmt.Errorf("%w: bad JSON string: %w", ErrInvalidSetting, err)
			}
		}
		v = str
	default:
		return nil, fmt.Errorf("%w: unsupported type %s", ErrInvalidSetting, f.Kind)
	}

	if err := f.validate(v); err != nil {
		return nil, err
	}
	return v, nil
}

func (f SettingField) validate(v any) error {
	str, ok := v.(string)
	if !ok {
		return nil
	}
	if str == "" {
		return fmt.Errorf("%w: value cannot be empty", ErrInvalidSetting)
	}
	if check := settingValidators[f.Key]; check != nil {
		if err := check(str); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidSetting, err)
		}
	}
	return nil
}

// Set parses and validates raw, assigns it to key and saves settings.json.
// Nothing is changed when the value is invalid.
func (s *Settings) Set(key, raw string) error {
	f, err := LookupSetting(key)
	if err != nil {
		return err
	}
	v, err := f.ParseValue(raw)
	if err != nil {
		return err
	}
	s.setField(f, v)
	return s.Save()
}

func (s *Settings) setField(f SettingField, v any) {
	reflect.ValueOf(s).Elem().Field(f.index).Set(reflect.ValueOf(v))
}

// Path returns the location of settings.json.
func (s *Settings) Path() string { return s.path }

// Save writes the settings to settings.json. The written file holds only
// valid values, so any problems found at load time are cleared.
func (s *Settings) Save() error {
	if err := s.save(); err != nil {
		return err
	}
	s.problems = nil
	return nil
}

// decodeSettings applies every valid entry of data to s and reports the rest.
// Invalid entries leave the field at its current value.
func decodeSettings(data []byte, s *Settings) []SettingProblem {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return []SettingProblem{{Err: fmt.Errorf("%w: %s", ErrInvalidSettingsFile, describeJSONError(data, err))}}
	}

	var problems []SettingProblem
	known := map[string]bool{}
	for _, f := range SettingFields() {
		known[f.Key] = true
		msg, ok := raw[f.Key]
		if !ok {
			continue
		}

		target := reflect.New(reflect.TypeOf(s).Elem().Field(f.index).Type)
		if err := json.Unmarshal(msg, target.Interface()); err != nil {
			problems = append(problems, SettingProblem{Key: f.Key, Err: fmt.Errorf("%w: expected a %s, got %s", ErrInvalidSetting, kindName(f.Kind), msg)})
			continue
		}
		v := target.Elem().Interface()
		if str, _ := v.(string); str == "" && f.Kind == reflect.String {
			// Empty strings are filled with defaults by ResetEmpty.
			continue
		}
		if err := f.validate(v); err != nil {
			problems = append(problems, SettingProblem{Key: f.Key, Err: err})
			continue
		}
		s.setField(f, v)
	}

	unknown := make([]string, 0)
	for key := range raw {
		if !known[key] {
			unknown = append(unknown, key)
		}
	}
	slices.Sort(unknown)
	for _, key := range unknown {
		problems = append(problems, SettingProblem{Key: key, Err: fmt.Errorf("%w (it will be dropped the next time settings are saved)", ErrUnknownSetting)})
	}
	return problems
}

// WarnProblems logs the invalid entries found when settings.json was loaded.
func (s *Settings) WarnProblems() {
	if len(s.problems) == 0 {
		return
	}
	log.Warn("settings.json has invalid entries; ZVM is using defaults for them. Run `zvm settings --check` for details or `zvm settings` to fix them.", "path", s.path)
	for _, p := range s.problems {
		log.Warn(p.String())
	}
}

// CheckSettingsFile reports every invalid entry in the settings file at path.
func CheckSettingsFile(path string) ([]SettingProblem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Settings
	return decodeSettings(data, &s), nil
}

func kindName(k reflect.Kind) string {
	if k == reflect.Bool {
		return "boolean (true or false)"
	}
	return k.String()
}

// describeJSONError adds a line and column to JSON syntax errors.
func describeJSONError(data []byte, err error) string {
	var offset int64 = -1
	var syntax *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &syntax):
		offset = syntax.Offset
	case errors.As(err, &typeErr):
		offset = typeErr.Offset
	}
	if offset < 0 {
		return err.Error()
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}
	// Offset counts the bytes read, including the offending one.
	if offset > 0 {
		offset--
	}
	before := data[:offset]
	line := bytes.Count(before, []byte("\n")) + 1
	col := int(offset) - bytes.LastIndexByte(before, '\n')
	return fmt.Sprintf("line %d, column %d: %v", line, col, err)
}
