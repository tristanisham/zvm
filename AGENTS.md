# Repository Guidelines

## Project Structure & Module Organization

ZVM is a Go CLI for installing and switching Zig and ZLS versions. `main.go`
defines commands and initializes the shared `cli.ZVM`. Core behavior lives in
`cli/`: installation and verification in `install.go`, version selection in
`use.go` and `resolve.go`, configuration in `config.go`, and self-upgrades in
`upgrade.go`. OS-specific links and metadata live under `cli/meta/`. Keep tests
beside their implementation as `*_test.go`. Installer entry points are
`install.sh` and `install.ps1`; `build.ts` drives Deno-based release builds.

## Build, Test, and Development Commands

- `go run . --help` runs the CLI from source.
- `go build -v .` builds the local `zvm` binary.
- `go build -tags noAutoUpgrades .` builds the package-manager variant without
  self-upgrade support.
- `go test -v ./...` runs the complete Go test suite.
- `go test -v ./cli -run TestResolveStable` runs one focused test.
- `go fmt ./...` formats Go files; run it before submitting changes.
- `go vet ./...` performs standard static checks. CI also runs
  `golangci-lint` with `errcheck` disabled.
- `deno run -A build.ts` creates cross-platform release artifacts.

## Coding Style & Naming Conventions

Follow `gofmt` output and normal Go indentation. Use camelCase names and keep
symbols unexported unless another package needs them. Put platform differences
in paired build-tagged files such as `link_unix.go` and `link_win.go`, rather
than scattering runtime OS checks. Preserve sentinel error behavior from
`cli/error.go`; wrap or join errors when callers need to inspect them. Keep
security-sensitive download verification intact unless the change explicitly
targets that behavior.

## Testing Guidelines

Use Go's `testing` package and name tests `TestBehavior`. Prefer table-driven
cases for parsers, version resolution, and platform-neutral logic. Add a
regression test for every bug fix. There is no numeric coverage requirement,
but all affected packages and `go test ./...` should pass. Tests must not write
to a user's real `~/.zvm`; use `t.TempDir()` and scoped environment variables.

## Commit & Pull Request Guidelines

Recent commits use short, imperative subjects, for example `Add support for
installing specific Zig development builds`. Keep each commit focused. Pull
requests should explain user-visible behavior, link the relevant issue, list
validation performed, and note platform-specific impact. Include terminal
output when CLI presentation changes. Disclose generative-AI assistance as
required by `CONTRIBUTING.MD`, and be prepared to explain every submitted
change.
