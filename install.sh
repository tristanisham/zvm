#!/usr/bin/env bash

# ZVM install script - v2.0.0 - ZVM: https://github.com/tristanisham/zvm

ARCH=$(uname -m)
OS=$(uname -s)

if [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi
if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
fi

# echo "Installing zvm-$OS-$ARCH"
zvm_installed_location=""
install_latest() {
    tmp_dir="$(mktemp -d)"
    echo "Downloading $1 to ${tmp_dir}"
    if [ "$(uname)" = "Darwin" ] || [ "$OS" = "Linux" ]; then
        if [ -x "$ZVM_BINARY_ON_DISK_LOCATION" ]; then # used for testing installs
            tar -cf "${tmp_dir}/zvm.tar" -C "${ZVM_BINARY_ON_DISK_LOCATION}" zvm
        elif command -v wget2 >/dev/null 2>&1; then
            echo "wget2 is installed. Using wget2..."
            wget2 -q --force-progress --max-redirect 5 -O zvm.tar "https://github.com/tristanisham/zvm/releases/latest/download/$1"
        elif command -v wget >/dev/null 2>&1; then
            echo "wget is installed. Using wget..."
            wget -q --show-progress --max-redirect 5 -O "${tmp_dir}/zvm.tar" "https://github.com/tristanisham/zvm/releases/latest/download/$1"
        else
            echo "wget is not installed. Using curl..."
            curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o "${tmp_dir}/zvm.tar"
        fi

        # Extract to temp dir and get installation paths
        tar -xf "${tmp_dir}/zvm.tar" -C "${tmp_dir}"
        chmod +x "${tmp_dir}/zvm"

        # Get installation paths from zvm env command
        env_output=$("$tmp_dir/zvm" env)
        self_dir=$(echo "$env_output" | grep -o '"self": "[^"]*"' | cut -d'"' -f4)
        echo "Installing zvm to ${self_dir}"
        mkdir -p "$self_dir"
        mv "${tmp_dir}/zvm" "${self_dir}/zvm"
        rm -rf "${tmp_dir}"
        zvm_installed_location="${self_dir}/zvm"
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
    # Get ZVM environment variables
    zvm_env=$("$zvm_installed_location" env)
    if [ $? -ne 0 ]; then
        echo "Failed to get ZVM environment variables"
        exit 1
    fi

    # Parse JSON output
    zvm_self=$(echo "$zvm_env" | grep -o '"self": "[^"]*' | cut -d'"' -f4)

    if [[ "$SHELL" == */fish ]]; then
        if ! fish -c "contains \"$zvm_self\" \$PATH"; then
            {
                echo
                echo "# ZVM"
                echo "set -gx PATH \$PATH \"$zvm_self\""
            } >>"$TARGET_FILE"
            echo "Restart fish or run 'source $TARGET_FILE' to start using ZVM in this shell!"
        else
            echo "zvm installed to a directory already in your PATH"
        fi
    else
        if ! [[ ":$PATH:" == *":${zvm_self}:"* ]]; then
            {
                echo
                echo "# ZVM"
                echo "export PATH=\"\$PATH:$zvm_self\""
            } >>"$TARGET_FILE"
            echo "Run 'source $TARGET_FILE' to start using ZVM in this shell!"
        else
            echo "zvm installed to a directory already in your PATH"
        fi
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
            echo -e "${GREEN}set -gx${NC} ${BLUE}PATH${NC}${GREEN} ${NC}${RED}\"\$PATH:${zvm_self}\"${NC}"
        else
            echo -e "${GREEN}export${NC} ${BLUE}PATH${NC}${GREEN}=${NC}${RED}\"\$PATH:${zvm_self}\"${NC}"
        fi
    else
        if [[ "$SHELL" == */fish ]]; then
        echo 'if not contains "$self_dir" $PATH'
        echo '    set -gx PATH $PATH "$self_dir"'
        echo 'end'
    else
        echo 'if [[ ":$PATH:" != *":$self_dir:"* ]]; then'
        echo '    export PATH="$PATH:$self_dir"'
        echo 'fi'
    fi
    fi
    echo "Run 'zvm i master' to install Zig"
fi
