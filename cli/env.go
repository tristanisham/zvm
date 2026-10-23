// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
)

// Shells that `zvm env` can emit code for.
const (
	ShellBash       = "bash"
	ShellZsh        = "zsh"
	ShellFish       = "fish"
	ShellPowerShell = "powershell"
)

// SupportedShells lists the values accepted by `zvm env --shell`.
var SupportedShells = []string{ShellBash, ShellZsh, ShellFish, ShellPowerShell}

// OnCdCommand is the hidden subcommand the --use-on-cd hook runs.
const OnCdCommand = "use-on-cd"

// NormalizeShell maps a shell name or executable path (e.g. "/bin/zsh",
// "pwsh.exe") to one of SupportedShells.
func NormalizeShell(name string) (string, error) {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(name)))
	base = strings.TrimSuffix(base, ".exe")
	// Login shells are sometimes reported as "-zsh".
	base = strings.TrimPrefix(base, "-")
	switch base {
	case "bash", "zsh", "fish":
		return base, nil
	case "pwsh", "powershell":
		return ShellPowerShell, nil
	}
	return "", fmt.Errorf("%w %q; pass --shell with one of: %s", ErrUnknownShell, name, strings.Join(SupportedShells, ", "))
}

// DetectShell guesses the shell from $SHELL. $SHELL is the login shell, not
// necessarily the running one, so callers should prefer an explicit --shell.
func DetectShell() (string, error) {
	sh := os.Getenv("SHELL")
	if sh == "" {
		return "", fmt.Errorf("%w: $SHELL is not set; pass --shell with one of: %s", ErrUnknownShell, strings.Join(SupportedShells, ", "))
	}
	return NormalizeShell(sh)
}

// Env creates a per-shell link for the calling shell and writes the code that
// activates it to w. shellPid is recorded in the link name for pruning.
func (z *ZVM) Env(w io.Writer, shell string, useOnCd bool, shellPid int) error {
	if !slices.Contains(SupportedShells, shell) {
		return fmt.Errorf("%w %q; pass --shell with one of: %s", ErrUnknownShell, shell, strings.Join(SupportedShells, ", "))
	}

	// Best effort: stale links from dead shells are harmless but accumulate.
	if n, err := PruneMultishells(MultishellMaxAge); err != nil {
		log.Debug("pruning per-shell links", "err", err)
	} else if n > 0 {
		log.Debug("pruned per-shell links", "count", n)
	}

	link, err := z.NewMultishellLink(shellPid)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, EnvScript(shell, link, useOnCd))
	return err
}

// EnvScript renders the shell code printed by `zvm env`.
func EnvScript(shell, link string, useOnCd bool) string {
	var b strings.Builder
	switch shell {
	case ShellBash, ShellZsh:
		q := posixQuote(link)
		fmt.Fprintf(&b, "export %s=%s\n", MultishellEnv, q)
		fmt.Fprintf(&b, "export PATH=%s:\"$PATH\"\n", q)
		if !useOnCd {
			break
		}
		if shell == ShellZsh {
			fmt.Fprintf(&b, "_zvm_use_on_cd() { command zvm %s; }\n", OnCdCommand)
			b.WriteString("autoload -U add-zsh-hook\n")
			b.WriteString("add-zsh-hook chpwd _zvm_use_on_cd\n")
			b.WriteString("_zvm_use_on_cd\n")
		} else {
			// bash has no chpwd hook; check before each prompt whether the
			// directory changed. This also catches pushd/popd and autocd.
			fmt.Fprintf(&b, "__zvm_use_on_cd() { if [ \"${__ZVM_LAST_PWD-}\" != \"$PWD\" ]; then __ZVM_LAST_PWD=\"$PWD\"; command zvm %s; fi; }\n", OnCdCommand)
			b.WriteString("case \";${PROMPT_COMMAND-};\" in *\";__zvm_use_on_cd;\"*) ;; *) PROMPT_COMMAND=\"__zvm_use_on_cd${PROMPT_COMMAND:+;$PROMPT_COMMAND}\" ;; esac\n")
			b.WriteString("__zvm_use_on_cd\n")
		}
	case ShellFish:
		q := fishQuote(link)
		fmt.Fprintf(&b, "set -gx %s %s\n", MultishellEnv, q)
		fmt.Fprintf(&b, "set -gx PATH %s $PATH\n", q)
		if useOnCd {
			b.WriteString("function _zvm_use_on_cd --on-variable PWD --description 'zvm: switch Zig on directory change'\n")
			b.WriteString("    status --is-command-substitution; and return\n")
			fmt.Fprintf(&b, "    command zvm %s\n", OnCdCommand)
			b.WriteString("end\n")
			b.WriteString("_zvm_use_on_cd\n")
		}
	case ShellPowerShell:
		q := psQuote(link)
		fmt.Fprintf(&b, "$env:%s = %s\n", MultishellEnv, q)
		fmt.Fprintf(&b, "$env:PATH = %s + [System.IO.Path]::PathSeparator + $env:PATH\n", q)
		if useOnCd {
			// Install the handler once per session: evaluating this twice would
			// otherwise chain zvm's handler to itself. PowerShell stores the
			// previous handler as an EventHandler delegate, so it is called with
			// .Invoke rather than &.
			b.WriteString("if (-not $global:__ZvmUseOnCdInstalled) {\n")
			b.WriteString("    $global:__ZvmUseOnCdInstalled = $true\n")
			b.WriteString("    $global:__ZvmPrevLocationChangedAction = $ExecutionContext.SessionState.InvokeCommand.LocationChangedAction\n")
			b.WriteString("    $ExecutionContext.SessionState.InvokeCommand.LocationChangedAction = {\n")
			b.WriteString("        param($zvmSource, $zvmArgs)\n")
			b.WriteString("        if ($global:__ZvmPrevLocationChangedAction) { $global:__ZvmPrevLocationChangedAction.Invoke($zvmSource, $zvmArgs) }\n")
			fmt.Fprintf(&b, "        & zvm %s\n", OnCdCommand)
			b.WriteString("    }\n")
			b.WriteString("}\n")
			fmt.Fprintf(&b, "& zvm %s\n", OnCdCommand)
		}
	}
	return b.String()
}

// posixQuote single-quotes s for bash and zsh.
func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// fishQuote single-quotes s for fish, where only \ and ' are special.
func fishQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return "'" + strings.ReplaceAll(s, "'", `\'`) + "'"
}

// psQuote single-quotes s for PowerShell, where ' is escaped by doubling.
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// statVersion stats an install directory through os.Root, so it can never
// resolve to anything outside baseDir.
func (z *ZVM) statVersion(ver string) (fs.FileInfo, error) {
	root, err := os.OpenRoot(z.baseDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.Stat(ver)
}

// UseOnCd is run by the --use-on-cd hook after every directory change. It
// looks for the nearest version request (see FindVersionSource) and, if one
// is found and installed, repoints this shell's link. It never prompts, never
// changes the global version, and prints nothing when nothing changes.
// Problems are reported as a single line on stderr and never as an error, so
// a cd is never disturbed.
func (z *ZVM) UseOnCd(stdout, stderr io.Writer) {
	link, ok, err := ActiveMultishellLink()
	if err != nil || !ok {
		log.Debug("use-on-cd: no per-shell link", "err", err)
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Debug("use-on-cd: getwd", "err", err)
		return
	}

	src, found, err := FindVersionSource(cwd)
	if err != nil || !found {
		log.Debug("use-on-cd: no version file", "cwd", cwd, "err", err)
		return
	}

	ver := src.Version
	if installed, err := z.GetInstalledVersions(); err == nil {
		if resolved, err := resolveVersionShorthand(ver, installed); err == nil {
			ver = resolved
		}
	}

	// The version comes from a file in the directory being entered, which
	// may be untrusted (a cloned repository). Never let it name anything but
	// an install directory inside baseDir.
	if err := validateVersionTarget(ver, "use"); err != nil {
		fmt.Fprintf(stderr, "zvm: ignoring invalid Zig version %q in %s\n", src.Version, src.File)
		return
	}
	if err := checkReservedVersion(ver, "use"); err != nil {
		fmt.Fprintf(stderr, "zvm: ignoring invalid Zig version %q in %s\n", src.Version, src.File)
		return
	}

	target := filepath.Join(z.baseDir, ver)
	if linkPointsTo(link, target) {
		return
	}

	if fi, err := z.statVersion(ver); err != nil || !fi.IsDir() {
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Debug("use-on-cd: stat", "target", target, "err", err)
		}
		fmt.Fprintf(stderr, "zvm: Zig %s (from %s) is not installed; run `zvm install %s`\n", src.Version, src.File, src.Version)
		return
	}

	if err := z.setShellLink(link, ver); err != nil {
		fmt.Fprintf(stderr, "zvm: could not switch to Zig %s: %v\n", ver, err)
		return
	}
	fmt.Fprintf(stdout, "Using Zig %s (from %s)\n", ver, src.File)
}
