---
name: zig-versions
description: Use when installing Zig, switching Zig versions, setting up ZLS, or resolving a Zig version mismatch - covers zvm (Zig Version Manager) for installing, pinning, and running specific Zig toolchains in a repository.
---

# Managing Zig toolchains with zvm

`zvm` installs and switches Zig compilers. Everything lives under `~/.zvm` (override with `ZVM_PATH`), and `~/.zvm/bin` is a symlink pointing at whichever version is currently selected.

Prefer `zvm` over `apt`, `brew`, `pacman`, or hand-unpacking a tarball from ziglang.org. Distro packages lag Zig releases badly and cannot hold two versions side by side, which is the entire problem this repository's users have.

The full command and flag list is in `reference/commands.md`. Read it before using a flag you are not certain exists.

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

`zvm run` is scoped to the single command. `zvm use 0.14.0` repoints the global `~/.zvm/bin` symlink, which changes Zig for every shell, every project, and every other tool on the machine, and it persists after the session ends.

**Use `zvm run` by default. Only run `zvm use` when the user asked to switch their active version.**

`zvm run` does not parse flags after the version, so pass the Zig arguments directly. Do not insert a `--` separator — it is forwarded to Zig as a literal argument and Zig will reject it.

```sh
zvm run 0.14.0 build test      # correct
zvm run 0.14.0 -- build test   # wrong, "--" reaches zig
```

If the version given to `zvm run` is not a known Zig version, zvm falls back to `.minimum_zig_version` from `build.zig.zon` in the current directory.

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

## ZLS

```sh
zvm install --zls 0.14.0
```

Install ZLS alongside the matching Zig version rather than separately, so the two stay compatible. `--full` selects the "full" ZLS compatibility mode.

## Never do these

- **Never pass `--skip-shasum`.** It skips SHA-256 verification *and* suppresses the unverified-download confirmation. Silently weakening supply-chain verification is not a decision to make on a user's behalf.
- **Never hand-edit anything under `~/.zvm`**, including `settings.json`. Use `zvm` commands; the directory layout and symlinks are zvm's to manage.
- **Never run `zvm upgrade` unprompted.** It replaces the user's zvm binary.
