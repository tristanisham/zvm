// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tristanisham/zvm/cli/meta"
)

// deadPid is above every platform's pid limit, so no process has it.
const deadPid = 2147483000

// setupMultishell gives the test its own base dir with fake installed
// versions and its own per-shell link directory.
func setupMultishell(t *testing.T, versions ...string) (*ZVM, string) {
	t.Helper()
	base := t.TempDir()
	for _, v := range versions {
		if err := os.MkdirAll(filepath.Join(base, v), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, v, "zig"), []byte("#!/bin/sh\necho "+v+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	msDir := filepath.Join(t.TempDir(), "multishells")
	t.Setenv(MultishellDirEnv, msDir)
	t.Setenv(MultishellEnv, "")
	return &ZVM{baseDir: base}, msDir
}

func readlink(t *testing.T, path string) string {
	t.Helper()
	got, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("readlink %s: %v", path, err)
	}
	return got
}

func TestMultishellDirPrecedence(t *testing.T) {
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Skip("no user cache dir:", err)
	}
	state := t.TempDir()
	override := t.TempDir()

	tests := []struct {
		name     string
		override string
		state    string
		want     string
	}{
		{name: "explicit override", override: override, state: state, want: override},
		{name: "XDG_STATE_HOME", state: state, want: filepath.Join(state, "zvm", "multishells")},
		{name: "relative XDG_STATE_HOME is ignored", state: "relative/state", want: filepath.Join(cache, "zvm", "multishells")},
		{name: "user cache dir fallback", want: filepath.Join(cache, "zvm", "multishells")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(MultishellDirEnv, tt.override)
			t.Setenv("XDG_STATE_HOME", tt.state)
			got, err := MultishellDir()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("MultishellDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewMultishellLinkTargetsGlobal(t *testing.T) {
	z, msDir := setupMultishell(t, "0.15.1")

	// No global version yet: the link follows ~/.zvm/bin.
	link, err := z.NewMultishellLink(123)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(link) != msDir || !multishellNameRe.MatchString(filepath.Base(link)) {
		t.Fatalf("unexpected link path %q", link)
	}
	if !strings.HasPrefix(filepath.Base(link), "123_") {
		t.Errorf("link %q does not record pid 123", link)
	}
	if got, want := readlink(t, link), filepath.Join(z.baseDir, "bin"); got != want {
		t.Errorf("link with no global version -> %q, want %q", got, want)
	}

	// With a global version: a snapshot of what bin points at.
	if err := z.setBin("0.15.1"); err != nil {
		t.Fatal(err)
	}
	link2, err := z.NewMultishellLink(123)
	if err != nil {
		t.Fatal(err)
	}
	if link2 == link {
		t.Fatal("two links got the same name")
	}
	if got, want := readlink(t, link2), filepath.Join(z.baseDir, "0.15.1"); got != want {
		t.Errorf("link -> %q, want %q", got, want)
	}

	// The link directory must never be listed as a version: creating it
	// inside the base dir is refused.
	t.Setenv(MultishellDirEnv, filepath.Join(z.baseDir, "multishells"))
	if _, err := z.NewMultishellLink(1); !errors.Is(err, ErrNoMultishellDir) {
		t.Fatalf("NewMultishellLink inside baseDir: err = %v, want ErrNoMultishellDir", err)
	}
	versions, err := z.GetInstalledVersions()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(versions, ",") != "0.15.1" {
		t.Errorf("GetInstalledVersions() = %v, want [0.15.1]", versions)
	}
}

func TestActiveMultishellLink(t *testing.T) {
	_, msDir := setupMultishell(t)
	if err := os.MkdirAll(msDir, 0o700); err != nil {
		t.Fatal(err)
	}
	valid := filepath.Join(msDir, "1_2_0123abcd")
	existingLink := filepath.Join(msDir, "1_3_0123abcd")
	if err := meta.Link(t.TempDir(), existingLink); err != nil {
		t.Fatal(err)
	}
	regular := filepath.Join(msDir, "1_4_0123abcd")
	if err := os.WriteFile(regular, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		env     string
		wantOK  bool
		wantErr error
	}{
		{name: "unset", env: "", wantOK: false},
		{name: "valid but missing link is recreated later", env: valid, wantOK: true},
		{name: "existing link", env: existingLink, wantOK: true},
		{name: "outside the multishell dir", env: filepath.Join(t.TempDir(), "1_2_0123abcd"), wantErr: ErrNotMultishellLink},
		{name: "foreign name", env: filepath.Join(msDir, "not-ours"), wantErr: ErrNotMultishellLink},
		{name: "relative path", env: "1_2_0123abcd", wantErr: ErrNotMultishellLink},
		{name: "regular file", env: regular, wantErr: ErrNotMultishellLink},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(MultishellEnv, tt.env)
			link, ok, err := ActiveMultishellLink()
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && link != filepath.Clean(tt.env) {
				t.Errorf("link = %q, want %q", link, tt.env)
			}
		})
	}
}

func TestUseVersionScope(t *testing.T) {
	tests := []struct {
		name         string
		withShell    bool
		global       bool
		wantPerShell bool
	}{
		{name: "no per-shell link changes global", withShell: false, global: false, wantPerShell: false},
		{name: "per-shell link changes only this shell", withShell: true, global: false, wantPerShell: true},
		{name: "--global changes global even with per-shell link", withShell: true, global: true, wantPerShell: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z, _ := setupMultishell(t, "0.15.1", "0.15.2")
			if err := z.setBin("0.15.1"); err != nil {
				t.Fatal(err)
			}
			bin := filepath.Join(z.baseDir, "bin")

			var link string
			if tt.withShell {
				var err error
				if link, err = z.NewMultishellLink(os.Getpid()); err != nil {
					t.Fatal(err)
				}
				t.Setenv(MultishellEnv, link)
			}

			got, perShell, err := z.UseVersion("0.15.2", tt.global)
			if err != nil {
				t.Fatal(err)
			}
			if got != "0.15.2" || perShell != tt.wantPerShell {
				t.Fatalf("UseVersion = (%q, %v), want (0.15.2, %v)", got, perShell, tt.wantPerShell)
			}

			wantGlobal, wantShell := "0.15.2", "0.15.1"
			if tt.wantPerShell {
				wantGlobal, wantShell = "0.15.1", "0.15.2"
			}
			if got := readlink(t, bin); got != filepath.Join(z.baseDir, wantGlobal) {
				t.Errorf("global bin -> %q, want %s", got, wantGlobal)
			}
			if link != "" {
				if got := readlink(t, link); got != filepath.Join(z.baseDir, wantShell) {
					t.Errorf("shell link -> %q, want %s", got, wantShell)
				}
			}
		})
	}
}

func TestPruneMultishells(t *testing.T) {
	type entry struct {
		name   string
		kind   string // "link", "file", "dir"
		pid    string
		remove bool
	}
	tests := []struct {
		name   string
		maxAge time.Duration
		entry  entry
	}{
		{name: "old link of dead shell is pruned", maxAge: 0, entry: entry{kind: "link", name: "%d_1_0123abcd", pid: "dead", remove: true}},
		{name: "old link of live shell is kept", maxAge: 0, entry: entry{kind: "link", name: "%d_1_0123abcd", pid: "self"}},
		{name: "young link of dead shell is kept", maxAge: time.Hour, entry: entry{kind: "link", name: "%d_1_0123abcd", pid: "dead"}},
		{name: "link with a foreign name is kept", maxAge: 0, entry: entry{kind: "link", name: "my-link"}},
		{name: "regular file with our name is kept", maxAge: 0, entry: entry{kind: "file", name: "%d_1_0123abcd", pid: "dead"}},
		{name: "directory with our name is kept", maxAge: 0, entry: entry{kind: "dir", name: "%d_1_0123abcd", pid: "dead"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, msDir := setupMultishell(t)
			if err := os.MkdirAll(msDir, 0o700); err != nil {
				t.Fatal(err)
			}

			// The link target holds a file that must survive any prune.
			target := t.TempDir()
			precious := filepath.Join(target, "zig")
			if err := os.WriteFile(precious, []byte("x"), 0o755); err != nil {
				t.Fatal(err)
			}

			name := tt.entry.name
			switch tt.entry.pid {
			case "dead":
				name = strings.Replace(name, "%d", strconv.Itoa(deadPid), 1)
			case "self":
				name = strings.Replace(name, "%d", strconv.Itoa(os.Getpid()), 1)
			}
			path := filepath.Join(msDir, name)
			switch tt.entry.kind {
			case "link":
				if err := meta.Link(target, path); err != nil {
					t.Fatal(err)
				}
			case "file":
				if err := os.WriteFile(path, nil, 0o644); err != nil {
					t.Fatal(err)
				}
			case "dir":
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			}

			n, err := PruneMultishells(tt.maxAge)
			if err != nil {
				t.Fatal(err)
			}
			_, statErr := os.Lstat(path)
			removed := errors.Is(statErr, os.ErrNotExist)
			if removed != tt.entry.remove {
				t.Errorf("removed = %v, want %v", removed, tt.entry.remove)
			}
			if (n == 1) != tt.entry.remove {
				t.Errorf("PruneMultishells returned %d", n)
			}
			if _, err := os.Stat(precious); err != nil {
				t.Errorf("link target contents were touched: %v", err)
			}
		})
	}
}

func TestPruneSkipsCurrentShell(t *testing.T) {
	_, msDir := setupMultishell(t)
	if err := os.MkdirAll(msDir, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(msDir, strconv.Itoa(deadPid)+"_1_0123abcd")
	if err := meta.Link(t.TempDir(), link); err != nil {
		t.Fatal(err)
	}
	t.Setenv(MultishellEnv, link)
	if n, err := PruneMultishells(0); err != nil || n != 0 {
		t.Fatalf("PruneMultishells = %d, %v; want 0, nil", n, err)
	}
}

func TestUseOnCd(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		start      string
		shellLink  bool
		wantShell  string // version the shell link should end up on
		wantStdout string
		wantStderr string
		dirs       []string // extra directories to create, relative to baseDir
	}{
		{
			name:       "switches to the nearest minimum_zig_version",
			files:      map[string]string{"proj/build.zig.zon": zon("0.15.2"), "proj/src/.keep": ""},
			start:      "proj/src",
			shellLink:  true,
			wantShell:  "0.15.2",
			wantStdout: "Using Zig 0.15.2 (from ",
		},
		{
			name:      "already on the requested version is silent",
			files:     map[string]string{"proj/build.zig.zon": zon("0.15.1")},
			start:     "proj",
			shellLink: true,
			wantShell: "0.15.1",
		},
		{
			name:       "shorthand resolves against installed versions",
			files:      map[string]string{"proj/build.zig": buildZigLock("0.15")},
			start:      "proj",
			shellLink:  true,
			wantShell:  "0.15.2",
			wantStdout: "Using Zig 0.15.2",
		},
		{
			name:       "uninstalled version prints a hint and changes nothing",
			files:      map[string]string{"proj/build.zig.zon": zon("0.11.0")},
			start:      "proj",
			shellLink:  true,
			wantShell:  "0.15.1",
			wantStderr: "zvm: Zig 0.11.0 (from ",
		},
		{
			name:      "no version file does nothing",
			files:     map[string]string{"proj/.keep": ""},
			start:     "proj",
			shellLink: true,
			wantShell: "0.15.1",
		},
		{
			name:       "zvm-lock cannot point outside baseDir",
			files:      map[string]string{"proj/build.zig": buildZigLock("x/../../evil")},
			dirs:       []string{"../evil"},
			start:      "proj",
			shellLink:  true,
			wantShell:  "0.15.1",
			wantStderr: "zvm: ignoring invalid Zig version",
		},
		{
			name:       "minimum_zig_version cannot point outside baseDir",
			files:      map[string]string{"proj/build.zig.zon": zon("x/../../evil")},
			dirs:       []string{"../evil"},
			start:      "proj",
			shellLink:  true,
			wantShell:  "0.15.1",
			wantStderr: "zvm: ignoring invalid Zig version",
		},
		{
			name:       "reserved entries are not versions",
			files:      map[string]string{"proj/build.zig": buildZigLock("self")},
			dirs:       []string{"self"},
			start:      "proj",
			shellLink:  true,
			wantShell:  "0.15.1",
			wantStderr: "zvm: ignoring invalid Zig version",
		},
		{
			name:  "without a per-shell link the global link is never touched",
			files: map[string]string{"proj/build.zig.zon": zon("0.15.2")},
			start: "proj",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z, _ := setupMultishell(t, "0.15.1", "0.15.2")
			if err := z.setBin("0.15.1"); err != nil {
				t.Fatal(err)
			}
			for _, d := range tt.dirs {
				if err := os.MkdirAll(filepath.Join(z.baseDir, filepath.FromSlash(d)), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			var link string
			if tt.shellLink {
				var err error
				if link, err = z.NewMultishellLink(os.Getpid()); err != nil {
					t.Fatal(err)
				}
				t.Setenv(MultishellEnv, link)
			}

			root := t.TempDir()
			writeTree(t, root, tt.files)
			t.Chdir(filepath.Join(root, filepath.FromSlash(tt.start)))

			var stdout, stderr bytes.Buffer
			z.UseOnCd(&stdout, &stderr)

			if !strings.HasPrefix(stdout.String(), tt.wantStdout) || (tt.wantStdout == "" && stdout.Len() != 0) {
				t.Errorf("stdout = %q, want prefix %q", stdout.String(), tt.wantStdout)
			}
			if !strings.HasPrefix(stderr.String(), tt.wantStderr) || (tt.wantStderr == "" && stderr.Len() != 0) {
				t.Errorf("stderr = %q, want prefix %q", stderr.String(), tt.wantStderr)
			}
			if strings.Count(stdout.String()+stderr.String(), "\n") > 1 {
				t.Errorf("hook printed more than one line: %q", stdout.String()+stderr.String())
			}
			if got := readlink(t, filepath.Join(z.baseDir, "bin")); got != filepath.Join(z.baseDir, "0.15.1") {
				t.Errorf("global bin changed to %q", got)
			}
			if link != "" {
				if got := readlink(t, link); got != filepath.Join(z.baseDir, tt.wantShell) {
					t.Errorf("shell link -> %q, want %s", got, tt.wantShell)
				}
			}
		})
	}
}

func TestNormalizeShell(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "/bin/bash", want: ShellBash},
		{in: "/usr/bin/zsh", want: ShellZsh},
		{in: "-zsh", want: ShellZsh},
		{in: "/opt/homebrew/bin/fish", want: ShellFish},
		{in: "pwsh", want: ShellPowerShell},
		{in: `C:\Program Files\PowerShell\7\pwsh.exe`, want: ShellPowerShell},
		{in: "powershell", want: ShellPowerShell},
		{in: "/bin/tcsh", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if runtime.GOOS != "windows" && strings.Contains(tt.in, `\`) {
				// filepath.Base does not split on backslashes off Windows.
				t.Skip("windows path")
			}
			got, err := NormalizeShell(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrUnknownShell) {
				t.Errorf("err = %v, want ErrUnknownShell", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestEnvScriptQuoting evaluates the generated bash code with a hostile link
// path and checks the variables come out intact.
func TestEnvScriptQuoting(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not installed")
	}
	link := `/tmp/it's a "dir" $HOME/1_2_0123abcd`
	script := EnvScript(ShellBash, link, false) + `printf '%s\n%s\n' "$ZVM_MULTISHELL_PATH" "${PATH%%:*}"`
	out, err := exec.Command(bash, "--norc", "--noprofile", "-c", script).CombinedOutput()
	if err != nil {
		t.Fatalf("bash: %v\n%s", err, out)
	}
	if want := link + "\n" + link + "\n"; string(out) != want {
		t.Errorf("got %q, want %q", out, want)
	}

	for _, shell := range SupportedShells {
		s := EnvScript(shell, link, true)
		if !strings.Contains(s, OnCdCommand) {
			t.Errorf("%s --use-on-cd script does not call %s", shell, OnCdCommand)
		}
		if strings.Contains(EnvScript(shell, link, false), OnCdCommand) {
			t.Errorf("%s script without --use-on-cd installs a hook", shell)
		}
	}
}

func TestSetShellLinkRejectsInvalidVersions(t *testing.T) {
	z, msDir := setupMultishell(t, "0.15.1")
	link := filepath.Join(msDir, "1_1_00000000")
	for _, ver := range []string{"x/../../evil", "../evil", `a\b`, "self", "bin", "settings.json", "store", "STORE", ""} {
		if err := z.setShellLink(link, ver); err == nil {
			t.Errorf("setShellLink(%q) succeeded, want an error", ver)
		}
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Errorf("link was created: %v", err)
	}
}

func TestMultishellDirRejectsRelativeOverride(t *testing.T) {
	t.Setenv(MultishellDirEnv, ".zvm-shells")
	if dir, err := MultishellDir(); !errors.Is(err, ErrNoMultishellDir) {
		t.Fatalf("MultishellDir() = %q, %v; want ErrNoMultishellDir", dir, err)
	}
}

func TestNewMultishellLinkRejectsDirInsideBaseDir(t *testing.T) {
	z, _ := setupMultishell(t, "0.15.1")
	for _, dir := range []string{z.baseDir, filepath.Join(z.baseDir, "state", "shells")} {
		t.Setenv(MultishellDirEnv, dir)
		if link, err := z.NewMultishellLink(os.Getpid()); !errors.Is(err, ErrNoMultishellDir) {
			t.Errorf("NewMultishellLink with %s = %q, %v; want ErrNoMultishellDir", dir, link, err)
		}
	}
	// A sibling whose name merely starts with the base dir's name is fine.
	t.Setenv(MultishellDirEnv, z.baseDir+"-shells")
	if _, err := z.NewMultishellLink(os.Getpid()); err != nil {
		t.Errorf("sibling dir rejected: %v", err)
	}
}

func TestEnvScriptPowerShellInstallsHookOnce(t *testing.T) {
	script := EnvScript(ShellPowerShell, `C:\links\1`, true)
	for _, want := range []string{"if (-not $global:__ZvmUseOnCdInstalled)", ".Invoke($zvmSource, $zvmArgs)"} {
		if !strings.Contains(script, want) {
			t.Errorf("PowerShell script missing %q:\n%s", want, script)
		}
	}
	if strings.Contains(script, "& $global:__ZvmPrevLocationChangedAction") {
		t.Errorf("PowerShell script still calls the previous handler with &:\n%s", script)
	}
}
