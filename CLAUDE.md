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

## Architecture

**Entry point:** `main.go` — defines the CLI app with `urfave/cli/v3`. A `Before` hook initializes a global `ZVM` instance that all command handlers share.

**Core type:** `cli.ZVM` (in `cli/config.go`) holds `baseDir` (default `~/.zvm`) and `Settings`. Initialized via `Initialize()` which creates the directory structure and loads `~/.zvm/settings.json`.

**Key flows:**
- **Install** (`cli/install.go`): fetches version map → downloads tarball → verifies minisign signature + SHA256 → extracts to `~/.zvm/<version>/`. Most complex file (~600 lines). Handles mirrors, ZLS co-installation, and platform-specific extraction.
- **Use** (`cli/use.go`): switches active version by symlinking `~/.zvm/bin` → `~/.zvm/<version>` via `meta.Link()`.
- **Upgrade** (`cli/upgrade.go`): self-upgrade from GitHub releases. Guarded by `!noAutoUpgrades` build tag.
- **Sync** (`cli/sync.go`): reads `build.zig.zon` for `//! zvm-lock: <version>` and switches to that version.

**Platform abstraction:** `cli/meta/link_unix.go` and `cli/meta/link_win.go` abstract symlinks (Unix) vs junctions (Windows). Similarly `cli/fileperms_unix.go` / `cli/fileperms_win.go` for permission checks.

**Version constant:** `cli/meta/version.go` — bump `VERSION` here for releases.

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
