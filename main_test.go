// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
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

func TestCompletionEnabled(t *testing.T) {
	if !zvmApp.EnableShellCompletion {
		t.Error("expected zvmApp.EnableShellCompletion to be true")
	}
	if zvmApp.ConfigureShellCompletionCommand == nil {
		t.Fatal("expected zvmApp.ConfigureShellCompletionCommand to be set")
	}
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
