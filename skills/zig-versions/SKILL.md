---
name: zig-versions
description: Use when managing Zig and ZLS toolchains with zvm, resolving a Zig version mismatch, or configuring ZVM and its disk usage.
---

# Managing Zig toolchains with zvm

`zvm` installs and switches Zig compilers. Installs and settings live under `~/.zvm` (override with `ZVM_PATH`), and `~/.zvm/bin` points at the global default. `zvm env` can give a shell its own version link outside that directory.

Prefer `zvm` over `apt`, `brew`, `pacman`, or hand-unpacking a tarball from ziglang.org. Distro packages lag Zig releases badly and cannot hold two versions side by side, which is the entire problem this repository's users have.

The full command and flag list is in [reference/commands.md](reference/commands.md). Read it before using a flag you are not certain exists; check the installed binary's help when its version differs from this checkout.

## Is zvm installed?

Probe first:

```sh
command -v zvm
```

If it is missing, **show the user the install command and ask** — do not run it yourself. The installer appends to shell profiles and modifies `PATH`, which is not a change to make on someone's behalf unprompted.

```sh
curl https://www.zvm.app/install.sh | bash   # Linux, macOS, BSD
irm "https://www.zvm.app/install.ps1" | iex  # Windows PowerShell
```

If the user declines, fall back to whatever the project already uses. Do not install Zig through a second package manager alongside an existing `zvm`.

## Running a specific version: prefer `run` over `use`

```sh
zvm run 0.14.0 build test
```

`zvm run` is scoped to the single command. In a shell initialized with `zvm env`, `zvm use 0.14.0` changes only that shell's link. Otherwise it changes the global `~/.zvm/bin` link. `zvm use --global 0.14.0` changes the default even in a per-shell session; existing per-shell links keep their own versions.

**Use `zvm run` by default. Only run `zvm use` when the user asked to switch their active version.**

`zvm run` does not parse flags after the version, so pass the Zig arguments directly. Do not insert a `--` separator — it is forwarded to Zig as a literal argument and Zig will reject it.

```sh
zvm run 0.14.0 build test      # correct
zvm run 0.14.0 -- build test   # wrong, "--" reaches zig
```

If the version given to `zvm run` is not a known Zig version, zvm falls back to `.minimum_zig_version` from `build.zig.zon` in the current directory.

## Per-shell versions

`zvm env --shell bash` prints shell code; it does not change the calling shell by itself. Use the shell the user is actually running. Add `--use-on-cd` when they want automatic switching from the nearest `build.zig` pin or `build.zig.zon` minimum version. The hook only switches to installed versions and never installs a missing one.

`ZVM_MULTISHELL_PATH` identifies the current shell's link. An override in `ZVM_MULTISHELL_DIR` must be absolute and outside the ZVM base directory. Do not edit shell startup files unless setting up shell integration is part of the request.

## Picking the right version for a repository

Check these, in order, before choosing a version:

1. **`//! zvm-lock: <version>` in `build.zig`** — zvm's own pin. Apply it with `zvm use --sync`, which reads that comment and switches to it. Note this is `build.zig`, *not* `build.zig.zon`.
2. **`.minimum_zig_version` in `build.zig.zon`** — a floor, not a pin. It says what the project needs at minimum, not what it was tested against.
3. **CI configuration** — usually the most honest signal of the version the project actually builds with.

Only fall back to `master` or `stable` when none of these exist.

## How zvm resolves version strings

- `0.15` resolves to the highest installed or available `0.15.x` patch.
- `.12` normalizes to `0.12`, then resolves as above.
- `stable` resolves to the highest non-dev, non-master release.
- `master` is the current development build and passes through unresolved.
- Aliases created with `zvm alias <name> <version>` also resolve.

Partial versions are safe to pass; you do not need to expand them yourself.

## Reading zvm output

`zvm ls-remote --json` is the machine-readable path. Use it whenever you need to select a version programmatically.

`zvm ls` prints a decorated table intended for humans. Read it, but do not build parsing logic on it. Color is stripped automatically when stdout is not a TTY, so piped output is already clean — do **not** pass `--color=false` to achieve that, because it prints an extra `Terminal color output: OFF` line that breaks naive matching.

## Non-interactive contexts

`zvm install` prompts for confirmation when a release has no published SHA-256 checksum, and `zvm use`/`zvm run` prompt before installing a version that is missing locally. With no stdin attached these prompts read EOF and are treated as "no", so the command declines safely rather than hanging.

If you need a version present without a prompt, install it explicitly first:

```sh
zvm install 0.14.0
```

## Configuration

Use non-interactive settings commands in agent workflows:

```sh
zvm settings .                       # all effective settings as JSON
zvm settings .useColor               # one JSON value
zvm settings -r .mirrorListUrl        # a raw string
zvm settings .alwaysInstallZLS true   # validate and persist one value
zvm settings --check                 # exit 1 for invalid file entries
```

The command accepts flat jq-style keys, including quoted forms; it does not implement arbitrary jq expressions or nested paths. No arguments opens an editor in a terminal and prints JSON otherwise; Plan 9 always uses JSON output.

Existing commands such as `zvm mirrorlist`, `zvm vmu`, and the global `--color` flag still work. Keep using them when they fit the task. `--color` persists the same `useColor` setting and prints a status line; it is not a flag for cleaning up machine-readable output.

Invalid entries use defaults without rewriting the user's file. Saving settings replaces invalid entries with effective values and drops unknown keys, so inspect reported problems before repairing the file. Do not clear invalid entries merely to silence the warning.

When adding or restructuring `Settings` fields in this repository, re-evaluate command discovery, path parsing, validation, the editor and their tests. They currently assume flat strings and booleans; nested fields need explicit support.

## ZLS

```sh
zvm install --zls 0.14.0
```

Install ZLS alongside the matching Zig version rather than separately, so the two stay compatible. `--full` selects the "full" ZLS compatibility mode.

## Sharing installed files

Use `zvm dedupe` when the user wants to reduce disk usage across installed versions. Pass version names to limit the operation:

```sh
zvm dedupe 0.15.1 0.15.2
zvm dedupe --auto on       # persist automatic dedupe after future installs
zvm dedupe --auto off
```

Auto-dedupe is off by default. Enabling it is a persistent configuration change, not part of an ordinary install. Dedupe uses hard links: editing a shared `lib/std` file in place changes every version linked to it. Do not dedupe a version the user intends to patch independently. Unsupported hard links are skipped without replacing the source files.

`zvm clean` and uninstall prune unreferenced store entries. `store` is reserved, not a version or alias target; leave its contents to ZVM.

## Never do these

- **Never pass `--skip-shasum`.** It skips SHA-256 verification *and* suppresses the unverified-download confirmation. Silently weakening supply-chain verification is not a decision to make on a user's behalf.
- **Never hand-edit anything under `~/.zvm`**, including `settings.json`. Use `zvm` commands; the directory layout and symlinks are zvm's to manage.
- **Never run `zvm upgrade` unprompted.** It replaces the user's zvm binary.
