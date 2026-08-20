# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ZVM (Zig Version Manager) is a CLI tool written in Go for installing and managing multiple Zig compiler versions. It also supports installing ZLS (Zig Language Server). Built with `urfave/cli/v3`.

## Build & Test Commands

```bash
go build -v .                    # Build the binary
go test -v ./...                 # Run all tests
go test -v ./cli/ -run TestName  # Run a single test
go vet ./...                     # Static analysis
go fmt ./...                     # Format code
```

Build without self-upgrade capability (for package manager distributions):
```bash
go build -tags noAutoUpgrades .
```

Package managers should also override the upgrade hint shown by `zvm upgrade`. The default in `main.go` is generic; inject a specific one (note the quotes — the value contains spaces and Go re-splits the `-ldflags` string):
```bash
go build -tags noAutoUpgrades \
  -ldflags "-s -w -X 'main.BuildUpgradeMessage=Use \`brew upgrade zvm\` to update ZVM.'" .
```

Deno tasks (see `deno.jsonc`):
```bash
deno task build          # cross-platform release artifacts into build/
deno task star-history   # regenerate the README star chart (needs GITHUB_TOKEN)
```

## Architecture

**Entry point:** `main.go` — defines the CLI app with `urfave/cli/v3`. A `Before` hook initializes a global `ZVM` instance that all command handlers share.

**Core type:** `cli.ZVM` (in `cli/config.go`) holds `baseDir` (default `~/.zvm`) and `Settings`. Initialized via `Initialize()` which creates the directory structure and loads `~/.zvm/settings.json`.

**Key flows:**
- **Install** (`cli/install.go`): fetches version map → downloads tarball → verifies minisign signature + SHA256 → extracts to `~/.zvm/<version>/`. Most complex file (~600 lines). Handles mirrors, ZLS co-installation, and platform-specific extraction.
- **Use** (`cli/use.go`): switches active version by symlinking `~/.zvm/bin` → `~/.zvm/<version>` via `meta.Link()`.
- **Upgrade** (`cli/upgrade.go`): self-upgrade from GitHub releases. Guarded by `!noAutoUpgrades` build tag.
- **Sync** (`cli/sync.go`): reads `build.zig` for `//! zvm-lock: <version>` and switches to that version. (Note: `build.zig`, not `build.zig.zon` — `.minimum_zig_version` in `build.zig.zon` is a separate fallback used by `cli/use.go:ExtractMinimumZigVersion`.)

**Platform abstraction:** `cli/meta/link_unix.go` and `cli/meta/link_win.go` abstract symlinks (Unix) vs junctions (Windows). Similarly `cli/fileperms_unix.go` / `cli/fileperms_win.go` for permission checks.

**Version constant:** `cli/meta/version.go` — bump `VERSION` here for releases.

## Packaging & Generated Files

- `installer/windows/` is a **build input**, not documentation — `.github/workflows/winget.yml` runs `dotnet build installer/windows/ZVM.wixproj`. Breaking `Package.wxs` breaks the MSI. It pins WiX 5.0.2 deliberately (WiX 6+ is under Open Source Maintenance Fee terms). Per-user MSI components must key off an HKCU registry value and register a `RemoveFolder`, or ICE38/ICE64 fail the build.
- `.github/assets/star-history-*.svg` are **generated** by `.github/workflows/star-history.yml`. Never hand-edit; run `deno task star-history`. They embed no timestamp, so an unchanged star count produces no commit.
- `docs/` is gitignored.

## Environment Variables

- `ZVM_PATH` — override default `~/.zvm` base directory
- `ZVM_DEBUG` — enable debug logging
- `ZVM_SET_CU` — disable background upgrade checker
- `ZVM_SKIP_TLS_VERIFY` — skip TLS verification for restricted networks

## Conventions

- Platform-specific code uses build tags (`//go:build windows`, `//go:build linux`, etc.) in paired files, not runtime switches.
- HTTP requests set `User-Agent: zvm <version>` and custom `X-Client-Os`/`X-Client-Arch` headers.
- Download integrity is verified with minisign signatures using Zig's public key, then SHA256 checksums.
- Errors are defined as sentinel values in `cli/error.go` and composed with `errors.Join()`.
- Tests use table-driven patterns with struct slices.
- `ZVM_INSTALL` is read only in `cli/upgrade.go`, behind `//go:build !noAutoUpgrades` — it is dead code in package-manager builds.
- Exercise the CLI against a scratch home rather than your real one: `env -i HOME=$(mktemp -d) PATH=/usr/bin:/bin ./zvm ls`. Any command that initializes creates `~/.zvm/settings.json` and `~/.zvm/self`.
- lipgloss strips ANSI automatically when stdout is not a TTY, so piped output is clean for assertions. Do not reach for `--color=false` to achieve this — it prints an extra `Terminal color output: OFF` line that breaks naive matching.
