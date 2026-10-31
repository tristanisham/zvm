// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/tristanisham/zvm/cli"
	"github.com/tristanisham/zvm/cli/meta"
	opts "github.com/urfave/cli/v3"
)

// Tags accepted by --silent. Each one names a part of ZVM that can be
// silenced for a single run.
const (
	// SilenceAUC silences the automatic upgrade checker: no network request
	// is made and no upgrade notice is printed.
	SilenceAUC = "auc"
)

// silenceTags lists every tag --silent accepts, in display order. A bare
// --silent silences all of them.
var silenceTags = []string{SilenceAUC}

// completionFlag is the argument urfave/cli appends to a shell completion
// request. It is only honored as the last argument.
const completionFlag = "--generate-shell-completion"

// silentFlag is the global --silent flag.
var silentFlag = &opts.GenericFlag{
	Name: "silent",
	Usage: "silence parts of ZVM for this run; bare --silent silences everything, " +
		"or name tags (comma-separated): auc = automatic upgrade checker",
	Value: &silenceValue{},
	// main reads --silent before Run (see parseSilence), and only scans the
	// global flags ahead of the subcommand. Local keeps urfave/cli from
	// accepting it after the subcommand, where it would parse but have no
	// effect.
	Local: true,
}

func init() {
	// The default help line would read "--silent value", suggesting
	// `--silent auc`. Tags must be attached with "=": a separate word is
	// taken as the subcommand.
	silentFlag.SetStringer(func(f opts.Flag) string {
		return "--silent[=TAGS]\t" + silentFlag.Usage
	})
}

// silenceValue is the value of --silent. It reports itself as a bool flag so
// urfave/cli accepts a bare --silent (which it passes to Set as "true")
// without consuming the next argument. Tags are given with an equals sign:
// --silent=auc. Several tags may be comma-separated or the flag repeated.
type silenceValue struct {
	tags map[string]bool
}

func (s *silenceValue) IsBoolFlag() bool { return true }

func (s *silenceValue) Set(val string) error {
	if s.tags == nil {
		s.tags = map[string]bool{}
	}

	// A bare --silent (or --silent=) arrives as "true": silence everything.
	if val == "true" {
		for _, tag := range silenceTags {
			s.tags[tag] = true
		}
		return nil
	}

	for _, tag := range strings.Split(val, ",") {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if !slices.Contains(silenceTags, tag) {
			return fmt.Errorf("%w: unknown --silent tag %q (valid: %s)", cli.ErrInvalidInput, tag, strings.Join(silenceTags, ", "))
		}
		s.tags[tag] = true
	}
	return nil
}

func (s *silenceValue) String() string {
	return strings.Join(s.list(), ",")
}

func (s *silenceValue) Get() any {
	return s.list()
}

// Has reports whether tag is silenced.
func (s *silenceValue) Has(tag string) bool {
	return s != nil && s.tags[tag]
}

func (s *silenceValue) list() []string {
	out := []string{}
	for _, tag := range silenceTags {
		if s.Has(tag) {
			out = append(out, tag)
		}
	}
	return out
}

// parseSilence reads --silent from the global flags in args (os.Args[1:]),
// stopping at the subcommand. main needs the answer before zvmApp.Run starts,
// because the upgrade check runs concurrently with the command. Invalid tags
// are ignored here; zvmApp.Run parses the same flag and reports them.
func parseSilence(args []string) *silenceValue {
	s := &silenceValue{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return s
		case arg == "--color" || arg == "-color":
			i++ // skip its value
		case strings.HasPrefix(arg, "-"):
			name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
			name, val, _ := strings.Cut(name, "=")
			if name != "silent" {
				continue
			}
			if val == "" {
				val = "true"
			}
			_ = s.Set(val)
		default:
			return s
		}
	}
	return s
}

// upgradeCheckDisabled reports whether the automatic upgrade checker should
// stay off for this run. args is os.Args[1:].
func upgradeCheckDisabled(args []string) bool {
	if _, ok := os.LookupEnv("ZVM_SET_CU"); ok {
		return true
	}

	if meta.NoAutoUpgrades {
		return true
	}

	if parseSilence(args).Has(SilenceAUC) {
		return true
	}

	// Shell completion output is read by the shell, so it must not wait on
	// the network or carry a notice. urfave/cli only treats the flag as a
	// completion request when it is the last argument.
	if len(args) > 0 && args[len(args)-1] == completionFlag {
		return true
	}

	// `zvm env` output is eval'd by the shell, the cd hook runs on every
	// directory change, and `zvm completion` prints a script that is
	// sourced: none may wait on the network or print a notice.
	// Only the subcommand counts: `zvm run 0.15.1 env` runs `zig env` and
	// should still check for upgrades.
	switch subcommand(args) {
	case "env", cli.OnCdCommand, "completion":
		return true
	}

	return false
}
