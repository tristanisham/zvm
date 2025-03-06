<p align="center">
  <img width="400px"  src ="https://github.com/tristanisham/zvm/assets/23124818/be5c3713-8aaf-4419-a1ae-acb29da36eae"/>
</p>

Zig Version Manager (zvm) is a tool for managing your
[Zig](https://ziglang.org/) installs. With std under heavy development and a
large feature roadmap, Zig is bound to continue changing. Breaking existing
builds, updating valid syntax, and introducing new features like a package
manager. While this is great for developers, it also can lead to headaches when
you need multiple versions of a language installed to compile your projects, or
a language gets updated frequently.

## Join our Community

- [Twitch](https://twitch.tv/atalocke)
- [Twitter|X](https://twitter.com/atalocke)

<a href="https://polar.sh/tristanisham"><picture><source media="(prefers-color-scheme: dark)" srcset="https://polar.sh/embed/subscribe.svg?org=tristanisham&label=Subscribe&darkmode"><img alt="Subscribe on Polar" src="https://polar.sh/embed/subscribe.svg?org=tristanisham&label=Subscribe"></picture></a>

# Installing ZVM

By default, ZVM will install to the following places:

macOS and Linux or other POSIX operating systems: `$HOME/.local/bin`
Windows: `%LOCALAPPDATA%\bin` (typically, this is something like `C:\Users\Username\AppData\Local`

In POSIX operating systems (Linux, FreeBSD, etc), the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/)
is honored, including the non-standard `XDG_BIN_HOME`. On all operating systems,
the installation path can be overridden specifically for ZVM using the
`ZVM_INSTALL` environment variable, which points directly to an installation
directory, or the `ZVM_PATH` environment variable, which will point to a directory
that zvm will use for all operations.

If you use `ZVM_PATH` to point to a directory, ZVM will download new ZIG versions
and symlink whichever version you specify with `zvm use` to `$ZVM_PATH/bin`.
You should add this folder to your path. After ZVM 0.2.3, when `ZVM_PATH` is
set, ZVM's installer will add ZVM to `$HOME/.zvm/self`.

Upgrades for ZVM will use the same logic to update itself

# Linux, BSD, MacOS, *nix

```sh
curl https://raw.githubusercontent.com/tristanisham/zvm/master/install.sh | bash
```

<!-- This script will **automatically append** ZVM's required environment variables (see below) to `~/.profile` or `~/.bashrc`. -->

<!-- If these files don't exist, append the following to your shell's startup script.
```sh
echo "# ZVM" >> $HOME/.profile
echo export ZVM_INSTALL="$HOME/.zvm/self" >> $HOME/.profile
echo export PATH="$PATH:$HOME/.zvm/bin" >> $HOME/.profile
echo export PATH="$PATH:$ZVM_INSTALL/" >> $HOME/.profile
``` -->

# Windows

## PowerShell

```ps1
irm https://raw.githubusercontent.com/tristanisham/zvm/master/install.ps1 | iex
```

## Command Prompt

```cmd
powershell -c "irm https://raw.githubusercontent.com/tristanisham/zvm/master/install.ps1 | iex"
```

# If You Have a Valid Version of Go Installed

```sh
go install -ldflags "-s -w" github.com/tristanisham/zvm@latest
```

## Manually

Please grab the
[latest release](https://github.com/tristanisham/zvm/releases/latest).

## Putting ZVM on your Path

ZVM requires a few directories to be on your `$PATH`. First, run `zvm env`, and
note the settings for `bin` and `self`. They will be the same directory in most
circumstances, but may be different. These are the directories to add to your
`$PATH` if they do not exist there already. If you don't know how to
update your environment variables permanently on Windows, you can follow
[this guide](https://www.computerhope.com/issues/ch000549.htm). Once you're in
the appropriate menu, add or append these directories to `$PATH`.

## Community Package

### AUR

`zvm` on the [Arch AUR](https://aur.archlinux.org/packages/zvm) is a
community-maintained package, and may be out of date.

# Why should I use ZVM?

While Zig is still pre-1.0 if you're going to stay up-to-date with the master
branch, you're going to be downloading Zig quite often. You could do it
manually, having to scoll around to find your appropriate version, decompress
it, and install it on your `$PATH`. Or, you could install ZVM and run
`zvm i master` every time you want to update. `zvm` is a static binary under a
permissive license. It supports more platforms than any other Zig version
manager. Its only dependency is `tar` on Unix-based systems. Whether you're on
Windows, MacOS, Linux, a flavor of BSD, or Plan 9 `zvm` will let you install,
switch between, and run multiple versions of Zig.

# Contributing and Notice

`zvm` is stable software. Pre-v1.0.0 any breaking changes will be clearly
labeled, and any commands potentially on the chopping block will print notice.
The program is under constant development, and the author is very willing to
work with contributors. **If you have any issues, ideas, or contributions you'd
like to suggest
[create a GitHub issue](https://github.com/tristanisham/zvm/issues/new/choose)**.

# How to use ZVM

## Install

```sh
zvm install <version> 
# Or
zvm i <version>
```

Use `install` or `i` to download a specific version of Zig. To install the
latest version, use "master".

```sh
# Example
zvm i master
```

### Force Install

As of `v0.7.6` ZVM will now skip downloading a version if it is already
installed. You can always force an install with the `--force` or `-f` flag.

```sh
zvm i --force master
```

You can also enable the old behavior by setting the new `alwaysForceInstall`
field to `true` in `~/.zvm/settings.json`.

### Install ZLS with ZVM

You can now install ZLS with your Zig download! To install ZLS with ZVM, simply
pass the `--zls` flag with `zvm i`. For example:

```sh
zvm i --zls master
```

#### Select ZLS compatibility mode

By default, ZVM will install a ZLS build, which can be used with the given Zig
version, but may not be able to build ZLS from source. If you want to use a ZLS
build, which can be built using the selected Zig version, pass the `--full` flag
with `zvm i --zls`. For example:

```sh
zvm i --zls --full master
```

> [!IMPORTANT]
> This does not apply to tagged releases, e.g.: `0.13.0`

## Switch between installed Zig versions

```sh
zvm use <version>
```

Use `use` to switch between versions of Zig.

```sh
# Example
zvm use master
```

## List installed Zig versions

```sh
# Example
zvm ls
```

Use `ls` to list all installed version of Zig.

### List all versions of Zig available

```sh
zvm ls --all
```

The `--all` flag will list the available verisons of Zig for download. Not the
versions locally installed.

### List set version maps

```sh
zvm ls --vmu
```

The `--vmu` flag will list set version maps for Zig and ZLS downloads.

## Uninstall a Zig version

```sh
# Example
zvm rm 0.10.0
```

Use `uninstall` or `rm` to remove an uninstalled version from your system.

## Upgrade your ZVM installation

As of `zvm v0.2.3` you can now upgrade your ZVM installation from, well, zvm.
Just run:

```sh
zvm upgrade
```

The latest version of ZVM should install on your machine, regardless of where
your binary lives (though if you have your binary in a privileged folder, you
may have to run this command with `sudo`).

Upgrades will honor the original installed directories. If you want to change
from a single directory in `$HOME/.zvm` to the platform-specific directory
scheme introduced in 0.8.6, the best path is to remove your existing zvm directory,
`ZVM_PATH`/`ZVM_INSTALL` environment variables, and reinstall.

## Clean up build artifacts

```sh
# Example
zvm clean
```

Use `clean` to remove build artifacts (Good if you're on Windows).

## Run installed version of Zig without switching your default

If you want to run a version of Zig without setting it as your default, the new
`run` command is your friend.

```sh
zig version
# 0.13.0

zvm run 0.11.0 version
# 0.11.0

zig version
# 0.13.0
```

This can be helpful if you want to test your project on a newer version of Zig
without having to switch between bins, or on alternative flavor of Zig.

## How to use with alternative VMUs

Make sure you switch your VMU before using `run`.

```sh
zvm vmu zig mach
run mach-latest version
# 0.14.0-dev.1911+3bf89f55c
```

If you would like to run the currently set Zig, please keep using the standard
`zig` command.

## Set Version Map Source

ZVM lets choose your vendor for Zig and ZLS. This is great if your company hosts
it's own internal fork of Zig, you prefer a different flavor of the language,
like Mach.

```sh
zvm vmu zig "https://machengine.org/zig/index.json" # Change the source ZVM pulls Zig release information from.

zvm vmu zls https://validurl.local/vmu.json
                                       # ZVM only supports schemas that match the offical version map schema. 
                                       # Run `vmu default` to reset your version map.

zvm vmu zig default # Resets back to default Zig releases.
zvm vmu zig mach # Sets ZVM to pull from Mach nominated Zig.

zvm vmu zls default # Resets back to default ZLS releases.
```

## Print program help

Print global help information by running:

```sh
zvm --help
```

Print help information about a specific command or subcommand.

```sh
zvm list --help
```

```
NAME:
   zvm list - list installed Zig versions. Flag `--all` to see remote options

USAGE:
   zvm list [command options] [arguments...]

OPTIONS:
   --all, -a   list remote Zig versions available for download, based on your version map (default: false)
   --vmu       list set version maps (default: false)
   --help, -h  show help
```

## Print program version

```sh
zvm --version
```

Prints the version of ZVM you have installed.

<hr>

## Option flags

### Color Toggle

Enable or disable colored ZVM output. No value toggles colors.

#### Enable

- on
- yes/y
- enabled
- true

#### Disabled

- off
- no/n
- disabled
- false

```sh
--color # Toggle ANSI color printing on or off for ZVM's output, i.e. --color=true
```

## Environment Variables

- `ZVM_DEBUG` enables DEBUG logging for your executable. This is meant for
  contributors and developers.
- `ZVM_SET_CU` Toggle the automatic upgrade checker. If you want to reenable the
  checker, just `uset ZVM_SET_CU`.
- `ZVM_PATH` replaces the default install location for ZVM Set the environment
  variable to the parent directory of where you've placed the `.zvm` directory.
- `ZVM_SKIP_TLS_VERIFY` Do you have problems using TLS in your evironment?
  Toggle off verifying TLS by setting this environment variable.
  - By default when this is enabled ZVM will print a warning. Set this variable
    to `no-warn` to silence this warning.

## Settings

ZVM has additional setting stored in `~/.zvm/settings.json`. You can manually
update version maps, toggle color support, and disable the automatic upgrade
checker here. All settings are also exposed as flags or environment variables.
This file is stateful, and ZVM will create it if it does not exist and utilizes
it for its operation.

## Please Consider Giving the Repo a Star ⭐

<!-- https://star-history.com/#tristanisham/zvm&Timeline -->
<a href="https://github.com/tristanisham/zvm">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=tristanisham/zvm&type=Timeline&theme=dark" />
    <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=tristanisham/zvm&type=Timeline" />
    <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=tristanisham/zvm&type=Timeline" />
  </picture>
</a>
