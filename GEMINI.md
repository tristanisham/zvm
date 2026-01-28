# Project Context: ZVM (Zig Version Manager)

## Project Overview

`zvm` is a cross-platform version manager for the Zig programming language,
written in Go. It allows users to install, switch between, and manage multiple
versions of Zig (including master/nightly builds and tagged releases) and ZLS
(Zig Language Server). It is designed to be simple, fast, and standalone, with
minimal dependencies (only `tar` on Unix systems).

## Architecture & Tech Stack

- **Language:** Go (v1.25.3)
- **Build System:** Deno (`build.ts`) is used for cross-platform compilation and
  bundling, though standard `go build` works for local development.
- **Key Libraries:**
  - `github.com/urfave/cli/v3`: CLI application framework.
  - `github.com/charmbracelet/log` & `lipgloss`: Styled terminal output and
    logging.
  - `github.com/jedisct1/go-minisign`: Verifying Zig signatures.

### Core Components

- **`main.go`**: The application entry point. Configures the CLI commands
  (`install`, `use`, `ls`, `clean`, `upgrade`, `vmu`) and flags.
- **`cli/` Package**: Contains the core business logic.
  - **`config.go`**: Handles initialization (`Initialize`), configuration
    loading, and the `ZVM` struct definition.
  - **`install.go`**: Logic for downloading, verifying (shasum/minisign), and
    extracting Zig versions.
  - **`use.go`**: Manages switching versions (symlinking).
  - **`ls.go`**: Listing installed and available remote versions.
  - **`settings.go`**: Manages `~/.zvm/settings.json`.
  - **`upgrade.go`**: Self-upgrade functionality.

## Building and Running

### Prerequisites

- Go 1.25+
- Deno (optional, for release builds)

### Development Commands

- **Run locally:**
  ```bash
  go run main.go [command]
  ```
- **Build locally:**
  ```bash
  go build -o zvm main.go
  ```
- **Run Tests:**
  ```bash
  go test ./...
  ```
- **Cross-Platform Build (Release):**
  ```bash
  deno run -A build.ts
  ```
  This script compiles `zvm` for Windows, Linux, macOS, *BSD, and Solaris,
  creating artifacts in the `build/` directory.

## Development Conventions

- **Formatting:** Strict adherence to `go fmt`.
- **Naming:** use `camelCase` for all variables, functions, and fields.
- **Visibility:** Default to private (lowercase) for functions/variables unless
  they _must_ be exported for external package use.
- **Environment:** The application expects to operate within `~/.zvm` (or
  `ZVM_PATH`) and uses `ZVM_INSTALL` and `PATH` modifications to function
  correctly.

## Key Files

- `main.go`: CLI definition and entry point.
- `cli/config.go`: Main `ZVM` struct and initialization logic.
- `build.ts`: Release build script (TypeScript/Deno).
- `CONTRIBUTING.MD`: Contribution guidelines and coding standards.
