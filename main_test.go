// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanisham/zvm/cli"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what
// was written. The urfave/cli/v3 completion generator writes directly to
// os.Stdout rather than the Command's Writer, so tests must intercept it here.
// fn receives the pipe's write end so callers can also point a Command's
// Writer at it — urfave caches Writer across Runs, so reassigning per-call
// avoids "file already closed" when the previous pipe has been torn down.
func captureStdout(t *testing.T, fn func(w io.Writer)) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn(w)
	_ = w.Close()
	return <-done
}

func setupAliasCommandTest(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(baseDir, "0.13.0"), 0755); err != nil {
		t.Fatalf("create installed version: %v", err)
	}
	t.Setenv("ZVM_PATH", baseDir)

	originalZVM := zvm
	zvm = *cli.Initialize()
	zvm.Settings.UseColor = false
	t.Cleanup(func() { zvm = originalZVM })
}

func TestAliasCommandLifecycle(t *testing.T) {
	setupAliasCommandTest(t)
	ctx := context.Background()

	if err := zvmApp.Run(ctx, []string{"zvm", "alias", "work", "0.13.0"}); err != nil {
		t.Fatalf("create alias: %v", err)
	}
	if got, ok, err := zvm.ResolveAlias(ctx, "work"); err != nil || !ok || got != "0.13.0" {
		t.Fatalf("created alias = %q, %v, %v; want %q, true, nil", got, ok, err, "0.13.0")
	}

	var printErr error
	got := captureStdout(t, func(w io.Writer) {
		zvmApp.Writer = w
		printErr = zvmApp.Run(ctx, []string{"zvm", "alias", "work"})
	})
	if printErr != nil {
		t.Fatalf("print alias: %v", printErr)
	}
	if !strings.Contains(got, "work 0.13.0") {
		t.Fatalf("alias output = %q; want alias name and version", got)
	}

	if err := zvmApp.Run(ctx, []string{"zvm", "alias", "--delete", "work"}); err != nil {
		t.Fatalf("delete alias: %v", err)
	}
	if _, ok, err := zvm.ResolveAlias(ctx, "work"); err != nil || ok {
		t.Fatalf("deleted alias still resolves: ok=%v, err=%v", ok, err)
	}

	if err := zvmApp.Run(ctx, []string{"zvm", "kv", "play", ".13"}); err != nil {
		t.Fatalf("create alias through kv shorthand: %v", err)
	}
	if got, err := resolveVersionArg(ctx, "play"); err != nil || got != "0.13.0" {
		t.Fatalf("resolveVersionArg(play) = %q, %v; want %q, nil", got, err, "0.13.0")
	}

	if err := zvmApp.Run(ctx, []string{"zvm", "alias", "--clear"}); err != nil {
		t.Fatalf("clear aliases: %v", err)
	}
	aliases, err := zvm.ListAliases(ctx)
	if err != nil {
		t.Fatalf("list aliases after clear: %v", err)
	}
	if len(aliases) != 0 {
		t.Fatalf("aliases after clear = %d; want 0", len(aliases))
	}
}

func TestAliasCommandRejectsInvalidArguments(t *testing.T) {
	setupAliasCommandTest(t)
	ctx := context.Background()

	err := zvmApp.Run(ctx, []string{"zvm", "alias", "missing", "9.9.9"})
	if !errors.Is(err, cli.ErrInvalidAliasValue) {
		t.Fatalf("non-installed version error = %v; want ErrInvalidAliasValue", err)
	}

	err = zvmApp.Run(ctx, []string{"zvm", "alias", "one", "two", "three"})
	if !errors.Is(err, cli.ErrInvalidInput) {
		t.Fatalf("too many arguments error = %v; want ErrInvalidInput", err)
	}

	err = zvmApp.Run(ctx, []string{"zvm", "alias", "--delete"})
	if !errors.Is(err, cli.ErrMissingArgument) {
		t.Fatalf("missing delete name error = %v; want ErrMissingArgument", err)
	}
}

func TestCompletionEnabled(t *testing.T) {
	if !zvmApp.EnableShellCompletion {
		t.Error("expected zvmApp.EnableShellCompletion to be true")
	}
	if zvmApp.ConfigureShellCompletionCommand == nil {
		t.Fatal("expected zvmApp.ConfigureShellCompletionCommand to be set")
	}
}

func TestInstallCommandHasSkipUseFlag(t *testing.T) {
	for _, command := range zvmApp.Commands {
		if command.Name != "install" {
			continue
		}

		for _, flag := range command.Flags {
			names := flag.Names()
			if len(names) == 2 && names[0] == "skip-use" && names[1] == "k" {
				return
			}
		}

		t.Fatal("install command does not define --skip-use with -k alias")
	}

	t.Fatal("install command is not registered")
}

func TestCompletionOutput(t *testing.T) {
	t.Setenv("ZVM_PATH", t.TempDir())

	tests := []struct {
		name  string
		shell string
		// Sentinel substring that should appear in the generated script.
		// Chosen to be stable across urfave/cli versions: the hidden flag
		// the completion script calls back with.
		signature string
		// Shell-specific signature, to confirm we got the right script.
		shellSig string
	}{
		{name: "bash", shell: "bash", signature: "--generate-shell-completion", shellSig: "__zvm_bash_autocomplete"},
		{name: "zsh", shell: "zsh", signature: "--generate-shell-completion", shellSig: "#compdef zvm"},
		{name: "fish", shell: "fish", signature: "--generate-shell-completion", shellSig: "commandline"},
		{name: "pwsh", shell: "pwsh", signature: "--generate-shell-completion", shellSig: "Register-ArgumentCompleter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var runErr error
			got := captureStdout(t, func(w io.Writer) {
				zvmApp.Writer = w
				// urfave/cli appends the completion subcommand during the first
				// Run and caches its Writer (set to os.Stdout at that moment).
				// Subsequent runs would write to a now-closed pipe unless we
				// refresh this subcommand's Writer too.
				for _, c := range zvmApp.Commands {
					if c.Name == "completion" {
						c.Writer = w
					}
				}
				runErr = zvmApp.Run(context.Background(), []string{"zvm", "completion", tt.shell})
			})
			if runErr != nil {
				t.Fatalf("unexpected error running completion %s: %v", tt.shell, runErr)
			}
			if got == "" {
				t.Fatal("expected non-empty completion script")
			}
			if !strings.Contains(got, tt.signature) {
				t.Errorf("completion script missing %q", tt.signature)
			}
			if !strings.Contains(got, tt.shellSig) {
				t.Errorf("completion script missing shell signature %q; got:\n%s", tt.shellSig, got)
			}
		})
	}
}

func TestFishCompletionIsStatic(t *testing.T) {
	t.Setenv("ZVM_PATH", t.TempDir())

	var runErr error
	got := captureStdout(t, func(w io.Writer) {
		zvmApp.Writer = w
		for _, c := range zvmApp.Commands {
			if c.Name == "completion" {
				c.Writer = w
			}
		}
		runErr = zvmApp.Run(context.Background(), []string{"zvm", "completion", "fish"})
	})
	if runErr != nil {
		t.Fatalf("unexpected error generating fish completion: %v", runErr)
	}
	if strings.Contains(got, "__zvm_perform_completion") {
		t.Fatal("fish completion should use static definitions, not dynamic completion")
	}
	if !strings.Contains(got, "complete -c zvm") {
		t.Fatal("fish completion is missing zvm completion definitions")
	}
	if !strings.Contains(got, "-l json") {
		t.Fatal("fish completion is missing the list-remote --json flag")
	}
}

func TestFormatHelpSectionHonorsColorSetting(t *testing.T) {
	original := zvm.Settings.UseColor
	t.Cleanup(func() { zvm.Settings.UseColor = original })

	zvm.Settings.UseColor = false
	if got := formatHelpSection("NAME:"); got != "NAME:" {
		t.Errorf("plain help section = %q, want NAME:", got)
	}

	zvm.Settings.UseColor = true
	if got := formatHelpSection("NAME:"); got == "NAME:" {
		t.Error("colored help section did not contain ANSI styling")
	}
}

func TestCompletionUnknownShell(t *testing.T) {
	t.Setenv("ZVM_PATH", t.TempDir())

	zvmApp.Writer = new(bytes.Buffer)
	zvmApp.ErrWriter = new(bytes.Buffer)

	err := zvmApp.Run(context.Background(), []string{"zvm", "completion", "tcsh"})
	if err == nil {
		t.Fatal("expected error for unknown shell, got nil")
	}
	if !strings.Contains(err.Error(), "tcsh") {
		t.Errorf("expected error to mention bad shell %q; got %v", "tcsh", err)
	}
}

func TestCompletionCommandVisible(t *testing.T) {
	t.Setenv("ZVM_PATH", t.TempDir())

	// The ConfigureShellCompletionCommand hook is applied during Run's setup.
	// A help invocation is enough to trigger it without producing a script.
	zvmApp.Writer = new(bytes.Buffer)
	if err := zvmApp.Run(context.Background(), []string{"zvm", "help"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, c := range zvmApp.Commands {
		if c.Name != "completion" {
			continue
		}
		found = true
		if c.Hidden {
			t.Error("expected completion command to be visible (Hidden=false)")
		}
		if c.Usage == "" {
			t.Error("expected completion command to have a Usage string")
		}
	}
	if !found {
		t.Fatal("completion command not registered on zvmApp")
	}
}
