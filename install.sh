#!/usr/bin/env bash

# ZVM install script - v0.2.0 - ZVM: https://github.com/tristanisham/zvm



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
        
        mkdir -p $HOME/.zvm/self
        tar -xf zvm.tar -C $HOME/.zvm/self
        rm "zvm.tar"
        
    elif [ $OS = "Linux" ]; then
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
        
        mkdir -p $HOME/.zvm/self
        tar -xf zvm.tar -C $HOME/.zvm/self
        rm "zvm.tar"
     elif [ $OS = "MINGW32_NT" ] || [ $OS == "MINGW64_NT" ]; then
        curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.zip
        # Additional extraction steps for Windows can be added here
    fi
}



if [ "$(uname)" = "Darwin" ]; then
    # Do something under Mac OS X platform
    install_latest "zvm-darwin-$ARCH.tar"
elif [ $OS = "Linux" ]; then
     # Do something under GNU/Linux platform
    install_latest "zvm-linux-$ARCH.tar"
elif [ $OS = "MINGW32_NT" ] || [ $OS == "MINGW64_NT" ]; then
    install_latest "zvm-windows-$ARCH.zip"
fi

# Determine the target file
if [ -f "$HOME/.profile" ]; then
    TARGET_FILE="$HOME/.profile"
elif [ -f "$HOME/.bashrc" ]; then
    TARGET_FILE="$HOME/.bashrc"
elif [ -f "$HOME/.zshenv" ]; then
    TARGET_FILE="$HOME/.zshenv"
elif [ -f "$HOME/.zshrc" ]; then
    TARGET_FILE="$HOME/.zshrc"
else
    TARGET_FILE=""
fi

if [ -n "$TARGET_FILE" ]; then
    # Check if variables are already present
    if grep -q 'ZVM_INSTALL' "$TARGET_FILE"; then
        echo "ZVM environment variables are already present in $TARGET_FILE"
        exit 0
    fi
    # Append the export statements to the TARGET_FILE
    echo "Adding ZVM environment variables to $TARGET_FILE"
    {
        echo
        echo "# ZVM"
        echo 'export ZVM_INSTALL="$HOME/.zvm/self"'
        echo 'export PATH="$PATH:$HOME/.zvm/bin"'
        echo 'export PATH="$PATH:$ZVM_INSTALL/"'
    } >> "$TARGET_FILE"
    echo "Run 'source $TARGET_FILE' to start using ZVM in this shell!"
    echo "Run 'zvm i master' to install Zig"
else
    echo
    echo "No ~/.profile, ~/.bashrc, ~/.zshenv or ~/.zshrc file found."
    echo "Run the following commands to set up ZVM environment variables in this session or append them to your shell's startup script:"
    echo
    if [[ "$TERM" == "xterm"* || "$TERM" == "screen"* || "$TERM" == "tmux"* ]]; then
        # Colors
        RED='\033[0;31m'   # For strings
        GREEN='\033[0;32m' # For commands
        BLUE='\033[0;34m'  # For variables
        NC='\033[0m'       # No Color

        echo -e "${GREEN}export${NC} ${BLUE}ZVM_INSTALL${NC}${GREEN}=${NC}${RED}\"\$HOME/.zvm/self\"${NC}"
        echo -e "${GREEN}export${NC} ${BLUE}PATH${NC}${GREEN}=${NC}${RED}\"\$PATH:\$HOME/.zvm/bin\"${NC}"
        echo -e "${GREEN}export${NC} ${BLUE}PATH${NC}${GREEN}=${NC}${RED}\"\$PATH:\$ZVM_INSTALL/\"${NC}"
        echo -e "Run 'zvm i master' to install Zig"
    else
        echo 'export ZVM_INSTALL="$HOME/.zvm/self"'
        echo 'export PATH="$PATH:$HOME/.zvm/bin"'
        echo 'export PATH="$PATH:$ZVM_INSTALL/"'
        echo "Run 'zvm i master' to install Zig"
    fi
fi
