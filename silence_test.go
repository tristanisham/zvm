// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/tristanisham/zvm/cli"
	"github.com/tristanisham/zvm/cli/meta"
)

func TestSilenceValueSet(t *testing.T) {
	tests := []struct {
		vals    []string
		want    []string
		wantErr bool
	}{
		{vals: []string{"true"}, want: silenceTags}, // bare --silent
		{vals: []string{"auc"}, want: []string{"auc"}},
		{vals: []string{"AUC"}, want: []string{"auc"}},
		{vals: []string{" auc "}, want: []string{"auc"}},
		{vals: []string{"auc", "auc"}, want: []string{"auc"}},
		{vals: []string{"auc,auc"}, want: []string{"auc"}},
		{vals: []string{"bogus"}, wantErr: true},
		{vals: []string{"auc,bogus"}, wantErr: true},
		{vals: []string{"auc,"}, wantErr: true},
		{vals: []string{"false"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.vals, "|"), func(t *testing.T) {
			s := &silenceValue{}
			var err error
			for _, v := range tt.vals {
				if err = s.Set(v); err != nil {
					break
				}
			}
			if tt.wantErr {
				if !errors.Is(err, cli.ErrInvalidInput) {
					t.Fatalf("Set(%q) error = %v, want ErrInvalidInput", tt.vals, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Set(%q) unexpected error: %v", tt.vals, err)
			}
			if got := s.Get().([]string); !slices.Equal(got, tt.want) {
				t.Fatalf("Set(%q) = %q, want %q", tt.vals, got, tt.want)
			}
		})
	}
}

func TestParseSilence(t *testing.T) {
	tests := []struct {
		args []string
		want bool // auc silenced
	}{
		{args: nil, want: false},
		{args: []string{"ls"}, want: false},
		{args: []string{"--silent", "ls"}, want: true},
		{args: []string{"-silent", "ls"}, want: true},
		{args: []string{"--silent=", "ls"}, want: true},
		{args: []string{"--silent=auc", "ls"}, want: true},
		{args: []string{"--silent=AUC", "ls"}, want: true},
		{args: []string{"--silent=bogus", "ls"}, want: false},
		{args: []string{"--color", "off", "--silent", "ls"}, want: true},
		{args: []string{"--color=off", "--silent=auc", "ls"}, want: true},
		// --silent is the value of --color here, as urfave/cli parses it.
		{args: []string{"--color", "--silent", "ls"}, want: false},
		// Only global flags before the subcommand count.
		{args: []string{"ls", "--silent"}, want: false},
		{args: []string{"run", "0.15.1", "build", "--silent"}, want: false},
		{args: []string{"--", "--silent"}, want: false},
		{args: []string{"--silentx", "ls"}, want: false},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			if got := parseSilence(tt.args).Has(SilenceAUC); got != tt.want {
				t.Fatalf("parseSilence(%q).Has(auc) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestUpgradeCheckDisabled(t *testing.T) {
	if meta.NoAutoUpgrades {
		t.Skip("the checker is always disabled in noAutoUpgrades builds")
	}
	// Make sure the environment does not disable the checker for every case.
	for _, env := range []string{"ZVM_DISABLE_AUC", "ZVM_SET_CU"} {
		t.Setenv(env, "")
		os.Unsetenv(env)
	}

	tests := []struct {
		args []string
		want bool
	}{
		{args: []string{"ls"}, want: false},
		{args: nil, want: false},
		{args: []string{"--help"}, want: false},
		{args: []string{"--silent", "ls"}, want: true},
		{args: []string{"--silent=auc", "install", "0.15.1"}, want: true},
		{args: []string{"env"}, want: true},
		{args: []string{"env", "--shell", "bash", "--use-on-cd"}, want: true},
		{args: []string{"--color=off", "env"}, want: true},
		{args: []string{cli.OnCdCommand}, want: true},
		{args: []string{"completion", "fish"}, want: true},
		{args: []string{"--generate-shell-completion"}, want: true},
		{args: []string{"install", "--generate-shell-completion"}, want: true},
		{args: []string{"use", "0.1", "--generate-shell-completion"}, want: true},
		// urfave/cli only completes when the flag is the last argument.
		{args: []string{"--generate-shell-completion", "ls"}, want: false},
		// `zig env` and `zig completion` through `zvm run` still check.
		{args: []string{"run", "0.15.1", "env"}, want: false},
		{args: []string{"run", "0.15.1", "completion"}, want: false},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			if got := upgradeCheckDisabled(tt.args); got != tt.want {
				t.Fatalf("upgradeCheckDisabled(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}

	for _, env := range []string{"ZVM_DISABLE_AUC", "ZVM_SET_CU"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv(env, "1")
			if !upgradeCheckDisabled([]string{"ls"}) {
				t.Fatalf("%s should disable the checker", env)
			}
		})
	}
}

// TestSilentFlagParsing checks that urfave/cli parses --silent the way
// parseSilence assumes: a bare flag takes no value, tags need "=", unknown
// tags are rejected, and the flag is not accepted after the subcommand.
func TestSilentFlagParsing(t *testing.T) {
	t.Setenv("ZVM_PATH", t.TempDir())
	zvmApp.Writer = new(bytes.Buffer)
	zvmApp.ErrWriter = new(bytes.Buffer)

	tests := []struct {
		args    []string
		wantErr bool
	}{
		{args: []string{"zvm", "--silent", "version"}},
		{args: []string{"zvm", "--silent=auc", "version"}},
		{args: []string{"zvm", "--silent=bogus", "version"}, wantErr: true},
		{args: []string{"zvm", "version", "--silent"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args[1:], " "), func(t *testing.T) {
			err := zvmApp.Run(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Run(%q) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}
