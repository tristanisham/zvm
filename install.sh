#!/usr/bin/env bash

# ZVM install script - v0.2.1 - ZVM: https://github.com/tristanisham/zvm

ARCH=$(uname -m)
OS=$(uname -s)

if [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi
if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
fi

# echo "Installing zvm-$OS-$ARCH"

install_latest() {
    echo -e "Downloading $1 in $(pwd)"
    if [ "$(uname)" = "Darwin" ]; then
        # Do something under MacOS platform
        if command -v wget >/dev/null 2>&1; then
            echo "wget is installed. Using wget..."
            wget -q --show-progress --max-redirect 5 -O zvm.tar "https://github.com/tristanisham/zvm/releases/latest/download/$1"
        else
            echo "wget is not installed. Using curl..."
            curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.tar
        fi

        mkdir -p "$HOME/.zvm/self"
        tar -xf zvm.tar -C "$HOME/.zvm/self"
        rm "zvm.tar"

    elif [ "$OS" = "Linux" ]; then
        # Do something under GNU/Linux platform
        if command -v wget2 >/dev/null 2>&1; then
            echo "wget2 is installed. Using wget2..."
            wget2 -q --force-progress --max-redirect 5 -O zvm.tar "https://github.com/tristanisham/zvm/releases/latest/download/$1"
        elif command -v wget >/dev/null 2>&1; then
            echo "wget is installed. Using wget..."
            wget -q --show-progress --max-redirect 5 -O zvm.tar "https://github.com/tristanisham/zvm/releases/latest/download/$1"
        else
            echo "wget is not installed. Using curl..."
            curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.tar
        fi

        mkdir -p "$HOME/.zvm/self"
        tar -xf zvm.tar -C "$HOME/.zvm/self"
        rm "zvm.tar"
    elif [ "$OS" = "MINGW32_NT" ] || [ "$OS" = "MINGW64_NT" ]; then
        curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.zip
        # Additional extraction steps for Windows can be added here
    fi
}

if [ "$(uname)" = "Darwin" ]; then
    install_latest "zvm-darwin-$ARCH.tar"
elif [ "$OS" = "Linux" ]; then
    install_latest "zvm-linux-$ARCH.tar"
elif [ "$OS" = "MINGW32_NT" ] || [ "$OS" = "MINGW64_NT" ]; then
    install_latest "zvm-windows-$ARCH.zip"
fi

###############################
# Determine the target file to update based on the user's shell.
# For Fish, we update ~/.config/fish/config.fish.
# For Zsh, we prefer .zshenv, .zprofile or .zshrc.
# Otherwise, we fallback to bash files (or any shell using .profile).

TARGET_FILE=""

if [[ "$SHELL" == */fish ]]; then
    TARGET_FILE="$HOME/.config/fish/config.fish"
elif [[ "$SHELL" == */zsh ]]; then
    if [ -f "$HOME/.zshenv" ]; then
        TARGET_FILE="$HOME/.zshenv"
    elif [ -f "$HOME/.zprofile" ]; then
        TARGET_FILE="$HOME/.zprofile"
    else
        TARGET_FILE="$HOME/.zshrc"
    fi
else
    if [ -f "$HOME/.bashrc" ]; then
        TARGET_FILE="$HOME/.bashrc"
    elif [ -f "$HOME/.profile" ]; then
        TARGET_FILE="$HOME/.profile"
    else
        TARGET_FILE=""
    fi
fi

###############################
# Append the ZVM environment variables if they are not already present.
if [ -n "$TARGET_FILE" ]; then
    if grep -q 'ZVM_INSTALL' "$TARGET_FILE"; then
        echo "ZVM environment variables are already present in $TARGET_FILE"
        exit 0
    fi
    echo "Adding ZVM environment variables to $TARGET_FILE"

    if [[ "$SHELL" == */fish ]]; then
        {
            echo
            echo "# ZVM"
            echo 'set -gx ZVM_INSTALL "$HOME/.zvm/self"'
            echo 'set -gx PATH $PATH "$HOME/.zvm/bin"'
            echo 'set -gx PATH $PATH "$ZVM_INSTALL/"'
        } >>"$TARGET_FILE"
        echo "Restart fish or run 'source $TARGET_FILE' to start using ZVM in this shell!"
    else
        {
            echo
            echo "# ZVM"
            echo 'export ZVM_INSTALL="$HOME/.zvm/self"'
            echo 'export PATH="$PATH:$HOME/.zvm/bin"'
            echo 'export PATH="$PATH:$ZVM_INSTALL/"'
        } >>"$TARGET_FILE"
        echo "Run 'source $TARGET_FILE' to start using ZVM in this shell!"
    fi
    echo "Run 'zvm i master' to install Zig"
else
    echo
    echo "No suitable shell startup file found."
    echo "Please add the following lines to your shell's startup script (or execute them in your current session):"
    if [[ "$TERM" == "xterm"* || "$TERM" == "screen"* || "$TERM" == "tmux"* ]]; then
        # Colors for pretty-printing
        RED='\033[0;31m'
        GREEN='\033[0;32m'
        BLUE='\033[0;34m'
        NC='\033[0m'
        if [[ "$SHELL" == */fish ]]; then
            echo -e "${GREEN}set -gx${NC} ${BLUE}ZVM_INSTALL${NC}${GREEN} ${NC}${RED}\"\$HOME/.zvm/self\"${NC}"
            echo -e "${GREEN}set -gx${NC} ${BLUE}PATH${NC}${GREEN} ${NC}${RED}\"\$PATH:\$HOME/.zvm/bin\"${NC}"
            echo -e "${GREEN}set -gx${NC} ${BLUE}PATH${NC}${GREEN} ${NC}${RED}\"\$PATH:\$ZVM_INSTALL/\"${NC}"
        else
            echo -e "${GREEN}export${NC} ${BLUE}ZVM_INSTALL${NC}${GREEN}=${NC}${RED}\"\$HOME/.zvm/self\"${NC}"
            echo -e "${GREEN}export${NC} ${BLUE}PATH${NC}${GREEN}=${NC}${RED}\"\$PATH:\$HOME/.zvm/bin\"${NC}"
            echo -e "${GREEN}export${NC} ${BLUE}PATH${NC}${GREEN}=${NC}${RED}\"\$PATH:\$ZVM_INSTALL/\"${NC}"
        fi
    else
        if [[ "$SHELL" == */fish ]]; then
            echo 'set -gx ZVM_INSTALL "$HOME/.zvm/self"'
            echo 'set -gx PATH $PATH "$HOME/.zvm/bin"'
            echo 'set -gx PATH $PATH "$ZVM_INSTALL/"'
        else
            echo 'export ZVM_INSTALL="$HOME/.zvm/self"'
            echo 'export PATH="$PATH:$HOME/.zvm/bin"'
            echo 'export PATH="$PATH:$ZVM_INSTALL/"'
        fi
    fi
    echo "Run 'zvm i master' to install Zig"
fi
