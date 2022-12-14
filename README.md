<p align="center">
  <img src ="https://user-images.githubusercontent.com/23124818/206966435-f5702a58-8b0e-4eb4-9dc4-b5e41ad27d8b.png"/>
</p
  
`zvm` (Zig Version Manager) is a tool for managing your Zig installs. With std under heavy development and a 
large feature roadmap, Zig is bound to continue changing. Breaking existing builds, updating 
valid sytax, and introducing new features like a package manager. While this is great for developers, it also
can lead to headaches when you need multiple versions of a language installed to compile your projects.

### Why should I use ZVM
`zvm` is a widly supported static binary under a permissive license. Unlike other tools, there are no local dependencies besides `tar`. It doesn't require linking to any libraries, and `zvm` supports a much wider install-base than any other Zig version manager. Whether you're on Windows, MacOS, Linux, a flavor of BSD, or Plan 9 `zvm` will let you install, switch between, and run multiple versions of Zig.

## Contributing and Notice
`zvm` is pre-alpha software, and makes no guarentees about its stability until at least v0.1.0. However, the program is under constant development, and the author is very willing to work with contributors. If you have any issues, ideas, or contributions you'd like to suggest create a GitHub issue. 

## Use
### Install
`zvm i <zig verion>`

`zmv install <zig verion>`

### Switching Between Versions
`zvm use <zig version>`
<hr>

### Cleaning Build Residue
`zvm clean` removed build artifiacts from Zig installs. The ABI works on Linux and Windows, but it's only neccessart to run on Windows. This command is also temporary and may not be included in future releases if the underlying problem it solves gets rectified.

## Installation
Just download one of the release binaries for your system. No external dependencies required. Just a static binary.

Add `~/.zvm/bin` to your path and `zvm` will automatically switch between versions of Zig for you.

### Community Package
#### AUR
https://aur.archlinux.org/packages/zvm
