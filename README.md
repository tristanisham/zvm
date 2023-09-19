<p align="center">
  <img width="100px"  src ="https://user-images.githubusercontent.com/23124818/206966435-f5702a58-8b0e-4eb4-9dc4-b5e41ad27d8b.png"/>
</p>

## Join our Community

- [Discord](https://discord.gg/NhaNhCMYX8)
- [Twitch](https://twitch.tv/atalocke)

<hr>

Zig Version Manager (zvm) is a tool for managing your Zig installs. With std
under heavy development and a large feature roadmap, Zig is bound to continue
changing. Breaking existing builds, updating valid sytax, and introducing new
features like a package manager. While this is great for developers, it also can
lead to headaches when you need multiple versions of a language installed to
compile your projects, or a language gets updated frequently.

# Installing ZVM

ZVM lives entirely in `$HOME/.zvm` on all platforms it supports. Inside of the
directory, ZVM will download new ZIG versions and symlink whichever version you
specify with `zvm use` to `$HOME/.zvm/bin`. You should add this folder to your
path. After ZVM 0.2.3, ZVMs installer will now add ZVM to `$HOME/.zvm/self`. You
should also add this directory as the environment variable `ZVM_INSTALL`. The
installer should handle this for you automatically if you're on *nix systems,
but you'll have to manually do this on Windows. You can then add
`ZVM_INSTALL to your path.`

If you don't want to use ZVM_INSTALL (like you already have ZVM in a place you
like), then ZVM will update the exact
executable you've called `upgrade` from.

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

If you're on Windows, please grab the
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
# Or
zvm --version
# Or
zvm -v
```

Prints the version of ZVM you have installed.

## Print program help

```sh
zvm help
# Or
zvm --help
# Or
zvm -h
```

<hr>

## Option flags

```sh
--nocolor, --nocolour   # Turns off ANSI color.
--color, --colour       # Toggles ANSI color.
--yescolor, --yescolour # Turns on ANSI color.
```
