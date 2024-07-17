#!/usr/bin/env bash

# ZVM install script - v0.1.5 - ZVM: https://github.com/tristanisham/zvm
ARCH=$(uname -m)
OS=$(uname -s)

post_install_setup() {
    if [[ "$TERM" == "xterm" || "$TERM" == "xterm-256color" || "$TERM" == "screen" || "$TERM" == "tmux" ]]; then
        # Colors
        RED='\033[0;31m'   # For errors
        GREEN='\033[0;32m' # For commands
        BLUE='\033[0;34m'  # For variables
        NC='\033[0m'       # No Color

        echo -e "${GREEN}To complete the setup, add the following lines to your .bashrc or .profile file:${NC}"

        echo
        echo -e "${BLUE}# ZVM${NC}"

        if [[ "$OS" == "Linux" ]]; then
            # Linux Instructions
            echo -e "export ZVM_PATH=\"${BLUE}\$XDG_DATA_HOME/zvm${NC}\""
            echo -e "export PATH=\"\${PATH}:${BLUE}\$ZVM_PATH/bin${NC}\""
            echo -e "${GREEN}Ensure that ${BLUE}\$HOME/.local/bin${NC} is in your PATH on Linux."
        elif [[ "$OS" == "Darwin" ]]; then
            # macOS Instructions
            echo -e "export ZVM_PATH=\"${BLUE}\$HOME/Library/zvm${NC}\""
            echo -e "export PATH=\"\${PATH}:${BLUE}\$ZVM_PATH/bin${NC}\""
            echo -e "${GREEN}Ensure that ${BLUE}\$HOME/bin${NC} is in your PATH on macOS."
        else
            # OS not supported
            echo -e "${RED}Your OS ($OS) is not supported by this script.${NC}"
        fi
    else
        echo "To complete the setup, add the following lines to your .bashrc or .profile file:"

        echo
        echo "# ZVM"

        if [[ "$OS" == "Linux" ]]; then
            # Linux Instructions
            echo "export ZVM_PATH=\"\$XDG_DATA_HOME/zvm\""
            echo "export PATH=\"\$PATH:\$ZVM_PATH/bin\""
            echo "Ensure that \$HOME/.local/bin is in your PATH on Linux."
        elif [[ "$OS" == "Darwin" ]]; then
            # macOS Instructions
            echo "export ZVM_PATH=\"\$HOME/Library/zvm\""
            echo "export PATH=\"\$PATH:\$ZVM_PATH/bin\""
            echo "Ensure that \$HOME/bin is in your PATH on macOS."
        else
            # OS not supported
            echo "Your OS ($OS) is not supported by this script."
        fi
    fi

    echo
    echo "Run 'zvm i master' to install Zig"
}

download_file() {
    local url="$1"
    local output_filename="$2"

    if command -v wget >/dev/null 2>&1; then
        echo "wget is installed. Using wget..."
        wget -q --show-progress --max-redirect 5 -O "$output_filename" "$url"
    elif command -v curl >/dev/null 2>&1; then
        echo "wget is not installed. Using curl..."
        curl -L --max-redirs 5 "$url" -o "$output_filename"
    elif command -v wget2 >/dev/null 2>&1; then
        echo "wget and curl are not installed. Using wget2..."
        wget2 -q --progress=bar --max-redirect 5 -O "$output_filename" "$url"
    else
        echo "Neither wget, curl, nor wget2 is installed. Please install one of these tools to proceed."
        return 1
    fi
}

install_zvm() {
    local install_dir
    local bin_dir

    case "$OS" in
    "Darwin")
        # MacOS platform
        install_dir="$HOME/Library/zvm/self"
        bin_dir="$HOME/bin"
        ;;
    "Linux")
        # GNU/Linux platform
        install_dir="${XDG_DATA_HOME:-$HOME/.local/share}/zvm/self"
        bin_dir="$HOME/.local/bin"
        ;;
    *)
        # Other OSes
        install_dir="$HOME/.zvm"
        bin_dir="$HOME/.zvm/bin"
        ;;
    esac

    # Check if the installation directory already exists
    if [ -d "$install_dir" ]; then
        echo "Error: zvm is already installed. Please remove the directory $install_dir to reinstall."
        return 1
    fi

    if ! download_file "https://github.com/tristanisham/zvm/releases/latest/download/$1" "zvm.zip"; then
        echo "Error: Failed to download zvm from https://github.com/tristanisham/zvm/releases/latest/download/$1"
        exit 1
    fi

    echo -e "Installing $1 in $install_dir"
    mkdir -p "$install_dir"
    tar -xf "zvm.zip" -C "$install_dir"
    mkdir -p "$bin_dir"
    ln -sf "$install_dir/zvm" "$bin_dir/zvm"
    rm "zvm.zip"

    post_install_setup

}

main() {
    case "$ARCH" in
    "aarch64")
        ARCH="arm64"
        ;;
    "x86_64")
        ARCH="amd64"
        ;;
    esac

    if [ "$OS" != "Darwin" ] && [ "$OS" != "Linux" ]; then
        echo "$OS-$ARCH not supported. If you are on Windows, please use install.ps1"
        return 1
    fi

    echo "Installing zvm-${OS,,}-$ARCH.tar"
    install_zvm "zvm-${OS,,}-$ARCH.tar"

}

main
