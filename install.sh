#!/usr/bin/env bash

# ZVM install script - v2.1.0 - ZVM: https://github.com/tristanisham/zvm
set -eu

NO_ENV=0
USE_XDG_SPEC=0
for arg in "$@"; do
    case "$arg" in
        --no-env)
            NO_ENV=1
            ;;
        --use-xdg-spec)
            USE_XDG_SPEC=1
            ;;
        *)
            echo "Unknown option: $arg" >&2
            exit 1
            ;;
    esac
done

ARCH=$(uname -m)
OS=$(uname -s)

if [ "$USE_XDG_SPEC" -eq 1 ] && [ "$OS" != "Linux" ]; then
    echo "--use-xdg-spec is only supported on Linux." >&2
    exit 1
fi

ZVM_SELF_DIR="$HOME/.zvm/self"

if [ "$USE_XDG_SPEC" -eq 1 ]; then
    zvm_xdg_config_home="${XDG_CONFIG_HOME:-$HOME/.config}"
    zvm_xdg_data_home="${XDG_DATA_HOME:-$HOME/.local/share}"
    zvm_xdg_state_home="${XDG_STATE_HOME:-$HOME/.local/state}"
    zvm_xdg_cache_home="${XDG_CACHE_HOME:-$HOME/.cache}"

    case "$zvm_xdg_config_home" in /*) ;; *) zvm_xdg_config_home="$HOME/.config" ;; esac
    case "$zvm_xdg_data_home" in /*) ;; *) zvm_xdg_data_home="$HOME/.local/share" ;; esac
    case "$zvm_xdg_state_home" in /*) ;; *) zvm_xdg_state_home="$HOME/.local/state" ;; esac
    case "$zvm_xdg_cache_home" in /*) ;; *) zvm_xdg_cache_home="$HOME/.cache" ;; esac

    ZVM_SELF_DIR="$HOME/.local/bin"
fi

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

        mkdir -p "$ZVM_SELF_DIR"
        tar -xf zvm.tar -C "$ZVM_SELF_DIR"
        rm "zvm.tar"

    elif [ "$OS" = "Linux" ]; then
        # Do something under GNU/Linux platform
        if command -v wget >/dev/null 2>&1; then
            echo "wget is installed. Using wget..."
            wget -q --show-progress --max-redirect 5 -O zvm.tar "https://github.com/tristanisham/zvm/releases/latest/download/$1"
            # TODO change so curl is checked and if fails, then do wget2. I don't like wget2's output. 
        elif command -v wget2 >/dev/null 2>&1; then
            echo "wget2 is installed. Using wget2..."
            wget2 -q --force-progress --max-redirect 5 -O zvm.tar "https://github.com/tristanisham/zvm/releases/latest/download/$1"
        else
            echo "wget or wget2 are not installed. Using curl..."
            curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.tar
        fi

        mkdir -p "$ZVM_SELF_DIR"
        tar -xf zvm.tar -C "$ZVM_SELF_DIR"
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

if [ "$USE_XDG_SPEC" -eq 1 ]; then
    mkdir -p \
        "$zvm_xdg_config_home/zvm" \
        "$zvm_xdg_data_home/zvm" \
        "$zvm_xdg_state_home/zvm" \
        "$zvm_xdg_cache_home/zvm"

    zvm_xdg_settings="$zvm_xdg_config_home/zvm/settings.json"
    if [ ! -f "$zvm_xdg_settings" ]; then
        printf '{\n    "useXDGSpec": true\n}\n' >"$zvm_xdg_settings"
        chmod 600 "$zvm_xdg_settings"
    elif ! grep -Eq '"useXDGSpec"[[:space:]]*:[[:space:]]*true' "$zvm_xdg_settings"; then
        echo "Warning: $zvm_xdg_settings does not enable useXDGSpec." >&2
        echo "Set \"useXDGSpec\" to true before running zvm." >&2
    fi
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

# Append the ZVM environment variables if they are not already present, unless --no-env is passed.
if [ "$NO_ENV" -eq 0 ]; then
    if [ -n "$TARGET_FILE" ]; then
        if [ "$USE_XDG_SPEC" -eq 1 ]; then
            if grep -q '\$HOME/.local/bin' "$TARGET_FILE"; then
                echo "ZVM's XDG executable directory is already present in $TARGET_FILE"
            else
                echo "Adding ZVM's XDG executable directory to $TARGET_FILE"
                if [[ "$SHELL" == */fish ]]; then
                    {
                        echo
                        echo "# ZVM"
                        echo 'if not contains "$HOME/.local/bin" $PATH'
                        echo '    set -gx PATH $PATH "$HOME/.local/bin"'
                        echo 'end'
                    } >>"$TARGET_FILE"
                    echo "Restart fish or run 'source $TARGET_FILE' to start using ZVM in this shell!"
                else
                    {
                        echo
                        echo "# ZVM"
                        echo 'case ":$PATH:" in'
                        echo '  *":$HOME/.local/bin:"*) ;;'
                        echo '  *) export PATH="$PATH:$HOME/.local/bin" ;;'
                        echo 'esac'
                    } >>"$TARGET_FILE"
                    echo "Run 'source $TARGET_FILE' to start using ZVM in this shell!"
                fi
            fi
        else
            if grep -q 'ZVM_INSTALL' "$TARGET_FILE"; then
                echo "ZVM environment variables are already present in $TARGET_FILE"
            else
                echo "Adding ZVM environment variables to $TARGET_FILE"
                if [[ "$SHELL" == */fish ]]; then
                    {
                        echo
                        echo "# ZVM"
                        echo 'set -gx ZVM_INSTALL "$HOME/.zvm/self"'
                        echo 'if test -d "$ZVM_INSTALL"'
                        echo '    set -gx PATH $PATH "$HOME/.zvm/bin"'
                        echo '    set -gx PATH $PATH "$ZVM_INSTALL"'
                        echo 'end'
                    } >>"$TARGET_FILE"
                    echo "Restart fish or run 'source $TARGET_FILE' to start using ZVM in this shell!"
                else
                    {
                        echo
                        echo "# ZVM"
                        echo 'export ZVM_INSTALL="$HOME/.zvm/self"'
                        echo 'if [ -d "$ZVM_INSTALL" ]; then'
                        echo '  export PATH="$PATH:$HOME/.zvm/bin"'
                        echo '  export PATH="$PATH:$ZVM_INSTALL"'
                        echo 'fi'
                    } >>"$TARGET_FILE"
                    echo "Run 'source $TARGET_FILE' to start using ZVM in this shell!"
                fi
            fi
        fi
        echo "Run 'zvm i master' to install Zig"
    else
        echo
        echo "No suitable shell startup file found."
        echo "Please add the following lines to your shell's startup script (or execute them in your current session):"
        if [ "$USE_XDG_SPEC" -eq 1 ]; then
            if [[ "$SHELL" == */fish ]]; then
                echo 'set -gx PATH $PATH "$HOME/.local/bin"'
            else
                echo 'export PATH="$PATH:$HOME/.local/bin"'
            fi
        elif [[ "$TERM" == "xterm"* || "$TERM" == "screen"* || "$TERM" == "tmux"* ]]; then
            # Colors for pretty-printing
            RED='\033[0;31m'
            GREEN='\033[0;32m'
            BLUE='\033[0;34m'
            NC='\033[0m'
            if [[ "$SHELL" == */fish ]]; then
                echo -e "${GREEN}set -gx${NC} ${BLUE}ZVM_INSTALL${NC}${GREEN} ${NC}${RED}\"\$HOME/.zvm/self\"${NC}"
                echo -e "${GREEN}if${NC} test -d ${RED}\"\$ZVM_INSTALL\"${NC}"
                echo -e "    ${GREEN}set -gx${NC} ${BLUE}PATH${NC}${GREEN} ${NC}${RED}\"\$PATH:\$HOME/.zvm/bin\"${NC}"
                echo -e "    ${GREEN}set -gx${NC} ${BLUE}PATH${NC}${GREEN} ${NC}${RED}\"\$PATH:\$ZVM_INSTALL\"${NC}"
                echo -e "${GREEN}end${NC}"
            else
                echo -e "${GREEN}export${NC} ${BLUE}ZVM_INSTALL${NC}${GREEN}=${NC}${RED}\"\$HOME/.zvm/self\"${NC}"
                echo -e "${GREEN}if${NC} [ -d ${RED}\"\$ZVM_INSTALL\"${NC} ]; ${GREEN}then${NC}"
                echo -e "  ${GREEN}export${NC} ${BLUE}PATH${NC}${GREEN}=${NC}${RED}\"\$PATH:\$HOME/.zvm/bin\"${NC}"
                echo -e "  ${GREEN}export${NC} ${BLUE}PATH${NC}${GREEN}=${NC}${RED}\"\$PATH:\$ZVM_INSTALL\"${NC}"
                echo -e "${GREEN}fi${NC}"
            fi
        else
            if [[ "$SHELL" == */fish ]]; then
                echo 'set -gx ZVM_INSTALL "$HOME/.zvm/self"'
                echo 'if test -d "$ZVM_INSTALL"'
                echo '    set -gx PATH $PATH "$HOME/.zvm/bin"'
                echo '    set -gx PATH $PATH "$ZVM_INSTALL"'
                echo 'end'
            else
                echo 'export ZVM_INSTALL="$HOME/.zvm/self"'
                echo 'if [ -d "$ZVM_INSTALL" ]; then'
                echo '  export PATH="$PATH:$HOME/.zvm/bin"'
                echo '  export PATH="$PATH:$ZVM_INSTALL"'
                echo 'fi'
            fi
        fi
        echo "Run 'zvm i master' to install Zig"
    fi
else
    echo "Skipping environment variable setup due to --no-env flag."
fi
