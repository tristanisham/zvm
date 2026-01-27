#!/usr/bin/env bash

# ZVM install script - v3.0.0 - ZVM: https://github.com/tristanisham/zvm
set -eu

ZVM_DIR="$HOME/.zvm"
ZVM_SELF="$ZVM_DIR/self"
GITHUB_URL="https://github.com/tristanisham/zvm/releases/latest/download"

# Detect platform
ARCH=$(uname -m)
OS=$(uname -s)

case "$ARCH" in
    x86_64)       ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

case "$OS" in
    Darwin) PLATFORM="darwin-$ARCH.tar" ;;
    Linux)  PLATFORM="linux-$ARCH.tar" ;;
    MINGW*|MSYS*) PLATFORM="windows-$ARCH.zip" ;;
    *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Download
echo "Installing zvm ($PLATFORM)..."
TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT

if command -v curl >/dev/null 2>&1; then
    curl -fSL "$GITHUB_URL/zvm-$PLATFORM" -o "$TMPFILE"
elif command -v wget >/dev/null 2>&1; then
    wget -q "$GITHUB_URL/zvm-$PLATFORM" -O "$TMPFILE"
elif command -v wget2 >/dev/null 2>&1; then
    wget2 -q "$GITHUB_URL/zvm-$PLATFORM" -O "$TMPFILE"
else
    echo "Error: curl, wget, or wget2 required" >&2
    exit 1
fi

# Extract
mkdir -p "$ZVM_SELF"
case "$PLATFORM" in
    *.tar) tar -xf "$TMPFILE" -C "$ZVM_SELF" ;;
    *.zip) unzip -qo "$TMPFILE" -d "$ZVM_SELF" ;;
esac

# Configure shell
case "$SHELL" in
    */fish)
        TARGET="$HOME/.config/fish/config.fish"
        ENV_BLOCK='
# ZVM
set -gx ZVM_INSTALL "$HOME/.zvm/self"
fish_add_path "$HOME/.zvm/bin" "$ZVM_INSTALL"'
        ;;
    */zsh)
        TARGET="${ZDOTDIR:-$HOME}/.zshrc"
        [[ -f "$HOME/.zshenv" ]] && TARGET="$HOME/.zshenv"
        ;&  # fall through
    *)
        TARGET="${TARGET:-$HOME/.bashrc}"
        [[ ! -f "$TARGET" ]] && TARGET="$HOME/.profile"
        ENV_BLOCK='
# ZVM
export ZVM_INSTALL="$HOME/.zvm/self"
export PATH="$PATH:$HOME/.zvm/bin:$ZVM_INSTALL"'
        ;;
esac

if [[ -f "$TARGET" ]] && grep -q 'ZVM_INSTALL' "$TARGET" 2>/dev/null; then
    echo "ZVM environment already configured in $TARGET"
else
    echo "$ENV_BLOCK" >> "$TARGET"
    echo "Added ZVM to $TARGET"
fi

echo "Done! Run 'source $TARGET' then 'zvm i master' to install Zig."