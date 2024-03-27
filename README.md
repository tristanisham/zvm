<p align="center">
  <img width="400px"  src ="https://github.com/tristanisham/zvm/assets/23124818/be5c3713-8aaf-4419-a1ae-acb29da36eae"/>
</p>

Zig Version Manager (zvm) is a tool for managing your Zig installs. With std
under heavy development and a large feature roadmap, Zig is bound to continue
changing. Breaking existing builds, updating valid syntax, and introducing new
features like a package manager. While this is great for developers, it also can
lead to headaches when you need multiple versions of a language installed to
compile your projects, or a language gets updated frequently.

## Join our Community

- [Twitch](https://twitch.tv/atalocke)
- [Twitter|X](https://twitter.com/atalocke)

<a href="https://polar.sh/tristanisham"><picture><source media="(prefers-color-scheme: dark)" srcset="https://polar.sh/embed/subscribe.svg?org=tristanisham&label=Subscribe&darkmode"><img alt="Subscribe on Polar" src="https://polar.sh/embed/subscribe.svg?org=tristanisham&label=Subscribe"></picture></a>

# Installing ZVM

ZVM lives entirely in `$HOME/.zvm` on all platforms it supports. Inside of the
directory, ZVM will download new ZIG versions and symlink whichever version you
specify with `zvm use` to `$HOME/.zvm/bin`. You should add this folder to your
path. After ZVM 0.2.3, ZVM's installer will now add ZVM to `$HOME/.zvm/self`.
You should also add this directory as the environment variable `ZVM_INSTALL`.
The installer should handle this for you automatically if you're on *nix
systems, but you'll have to manually do this on Windows. You can then add
`ZVM_INSTALL to your path.`

If you don't want to use ZVM_INSTALL (like you already have ZVM in a place you
like), then ZVM will update the exact executable you've called `upgrade` from.

# Linux, BSD, MacOS, *nix

```sh
curl https://raw.githubusercontent.com/tristanisham/zvm/master/install.sh | bash
```

Then add ZVM's directories to your `$PATH`

```sh
echo "# ZVM" >> $HOME/.profile
echo export ZVM_INSTALL="$HOME/.zvm/self" >> $HOME/.profile
echo export PATH="$PATH:$HOME/.zvm/bin" >> $HOME/.profile
echo export PATH="$PATH:$ZVM_INSTALL/" >> $HOME/.profile
```

# Windows

## PowerShell

```ps1
irm https://raw.githubusercontent.com/tristanisham/zvm/master/install.ps1 | iex
```

## Command Prompt

```cmd
powershell -c "irm https://raw.githubusercontent.com/tristanisham/zvm/master/install.ps1 | iex"
```

## Manually

Please grab the
[latest release](https://github.com/tristanisham/zvm/releases/latest).

## Putting ZVM on your Path

ZVM requires a few directories to be on your `$PATH`. If you don't know how to
update your environment variables perminantly on Windows, you can follow
[this guide](https://www.computerhope.com/issues/ch000549.htm). Once you're in
the appropriate menu, add or append to the following environment variables:

Add

- ZVM_INSTALL: `%USERPROFILE%\.zvm\self`

Append

- PATH: `%USERPROFILE%\.zvm\bin`
- PATH: `%ZVM_INSTALL%`

## Community Package

### AUR

`zvm` on the [Arch AUR](https://aur.archlinux.org/packages/zvm) is a community
maintained package, and may be out of date.

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

### Install ZLS with ZVM

You can now install ZLS with your Zig download! To install ZLS with ZVM, simply
pass the `-D=zls` flag with `zvm i`. For example:

```sh
zvm i -D=zls master
```

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
your binary lives (though if you have your binary in a privaledged folder, you
may have to run this command with `sudo`).

## Clean up build artifacts

```sh
# Example
zvm clean
```

Use `clean` to remove build artifacts (Good if you're on Windows).

## Print program version

```sh
zvm version
zvm --version
```

Prints the version of ZVM you have installed.

## Print program help

```sh
zvm help
```

<hr>

## Option flags

### Color Toggle

```sh
-color # Turn ANSI color printing on or off for ZVM's output, i.e. -color=true
```

### Version Map Source

```sh
-vmu="https://validurl.local/vmu.json" # Change the source ZVM pulls Zig release information from. Good for self-hosted Zig CDNs.
                                       # ZVM only supports schemas that match the offical version map schema. 
                                       # Run `-vmu=default` to reset your version map.

-vmu default # Resets back to default Zig releases.
-vmu mach # Sets ZVM to pull from Mach nominated Zig.
```

## Remember to Star the Repo

<a href="https://star-history.com/#tristanisham/zvm&Timeline">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=tristanisham/zvm&type=Timeline&theme=dark" />
    <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=tristanisham/zvm&type=Timeline" />
    <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=tristanisham/zvm&type=Timeline" />
  </picture>
</a>
