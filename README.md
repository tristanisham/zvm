<p align="center">
  <img width="100px"  src ="https://user-images.githubusercontent.com/23124818/206966435-f5702a58-8b0e-4eb4-9dc4-b5e41ad27d8b.png"/>
</p

`Zig Version Manager (zvm) is a tool for managing your Zig installs. With std
under heavy development and a large feature roadmap, Zig is bound to continue
changing. Breaking existing builds, updating valid sytax, and introducing new
features like a package manager. While this is great for developers, it also can
lead to headaches when you need multiple versions of a language installed to
compile your projects, or a language gets updated frequently.

# Installing ZVM

```sh
curl https://raw.githubusercontent.com/tristanisham/zvm/master/install.sh | bash
```

If you're on Windows, please grab the
[latest release](https://github.com/tristanisham/zvm/releases/latest).

## Community Package

### AUR

`zvm` on the [Arch AUR](https://aur.archlinux.org/packages/zvm) is a community
maintained package, and may be out of date.

# Why should I use ZVM

While Zig is still pre-1.0 if you're going to stay up-to-date with the master
branch, you're going to be downloading Zig quite often. You could do it
manually, having to scoll around to find your appropriate version, decompress
it, and install it on your `$PATH`. Or, you could install ZVM and run
`zvm i master` every time you want to update. `zvm` is a static binary under a
permissive license. It supports more platforms than any other Zig version
manager. It's only dependency is `tar` on Unix-based systems. Whether you're on
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
