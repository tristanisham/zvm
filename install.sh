#!/usr/bin/env bash

# ZVM install script - v0.1.5 - ZVM: https://github.com/tristanisham/zvm

ARCH=$(uname -m)
OS=$(uname -s)

if [ $ARCH = "aarch64" ]; then
    ARCH="arm64"
fi
if [ $ARCH = "x86_64" ]; then
    ARCH="amd64"
fi

# echo "Installing zvm-$OS-$ARCH"

install_latest() {
    if [ "$(uname)" = "Darwin" ]; then
        # MacOS platform
        install_dir="$HOME/Library/zvm/self"
        bin_dir="$HOME/bin"
    elif [ "$OS" = "Linux" ]; then
        # GNU/Linux platform
        install_dir="$XDG_DATA_HOME/zvm/self"
        bin_dir="$HOME/.local/bin"
    else
        # Other OSes
        install_dir="$HOME/.zvm"
        bin_dir="$HOME/.zvm/bin"
    fi

    echo -e "Installing $1 in $install_dir"

    if [ "$(uname)" = "Darwin" ]; then
        # MacOS platform
        if command -v wget >/dev/null 2>&1; then
            echo "wget is installed. Using wget..."
            wget -q --show-progress --max-redirect 5 -O zvm.tar "https://github.com/tristanisham/zvm/releases/latest/download/$1"
        else
            echo "wget is not installed. Using curl..."
            curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.tar
        fi

        mkdir -p "$install_dir"
        tar -xf zvm.tar -C "$install_dir"
        ln -s "$install_dir/zvm" "$bin_dir"
        rm "zvm.tar"

    elif [ "$OS" = "Linux" ]; then
        # GNU/Linux platform
        if command -v wget >/dev/null 2>&1; then
            echo "wget is installed. Using wget..."
            wget -q --show-progress --max-redirect 5 -O zvm.tar "https://github.com/tristanisham/zvm/releases/latest/download/$1"
        else
            echo "wget is not installed. Using curl..."
            curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.tar
        fi

        mkdir -p "$install_dir"
        tar -xf zvm.tar -C "$install_dir"
        ln -s "$install_dir/zvm" "$bin_dir"
        rm "zvm.tar"

    elif [ "$OS" = "MINGW32_NT" ]; then
        # 32 bits Windows NT platform
        curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.zip
        # Add extraction and installation commands as needed

    elif [ "$OS" = "MINGW64_NT" ]; then
        # 64 bits Windows NT platform
        curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.zip
        # Add extraction and installation commands as needed
    fi
}

if [ "$(uname)" = "Darwin" ]; then
    # Do something under Mac OS X platform
    install_latest "zvm-darwin-$ARCH.tar"
elif [ $OS = "Linux" ]; then
    # Do something under GNU/Linux platform
    install_latest "zvm-linux-$ARCH.tar"
elif [ $OS = "MINGW32_NT" ]; then
    # Do something under 32 bits Windows NT platform
    install_latest "zvm-windows-$ARCH.zip"
elif [ $OS == "MINGW64_NT" ]; then
    # Do something under 64 bits Windows NT platform
    install_latest "zvm-windows-$ARCH.zip"
fi

echo
echo "Run the following commands to put ZVM on your path via $HOME/.profile"
echo
# Check if TERM is set to a value that typically supports colors
if [[ "$TERM" == "xterm" || "$TERM" == "xterm-256color" || "$TERM" == "screen" || "$TERM" == "tmux" ]]; then
    # Colors
    RED='\033[0;31m'   # For strings
    GREEN='\033[0;32m' # For commands
    BLUE='\033[0;34m'  # For variables
    NC='\033[0m'       # No Color

    echo -e "${GREEN}echo${NC} ${RED}\"# ZVM\"${NC} ${GREEN}>>${NC} ${BLUE}\$HOME/.profile${NC}"
    echo -e "${GREEN}echo${NC} ${RED}'export ZVM_PATH=\"\$XDG_DATA_HOME/zvm\"'${NC} ${GREEN}>>${NC} ${BLUE}\$HOME/.profile${NC} ${GREEN}# Linux${NC}"
    echo -e "${GREEN}echo${NC} ${RED}'export ZVM_PATH=\"\$HOME/Library/zvm\"'${NC} ${GREEN}>>${NC} ${BLUE}\$HOME/.profile${NC} ${GREEN}# macOS${NC}"

    echo -e "Make sure ${RED}\$HOME/.local/bin${NC} is in your PATH on Linux."
    echo -e "Make sure ${RED}\$HOME/bin${NC} is in your PATH on macOS."

    echo "Run 'zvm i master' to install Zig"
else
    echo 'echo "# ZVM" >> $HOME/.profile'
    echo 'echo '\''export ZVM_PATH="$XDG_DATA_HOME/zvm"'\'' >> $HOME/.profile # Linux'
    echo 'echo '\''export ZVM_PATH="$HOME/Library/zvm"'\'' >> $HOME/.profile # macOS'

    echo "Make sure \$HOME/.local/bin is in your PATH on Linux."
    echo "Make sure \$HOME/bin is in your PATH on macOS."

    echo "Run 'zvm i master' to install Zig"
fi

echo
