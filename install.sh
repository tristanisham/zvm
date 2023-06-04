#!/usr/bin/env bash

# ZVM install script - v0.1.5 - ZVM: https://github.com/tristanisham/zvm



ARCH=$(uname -m)
OS=$(uname -s)


if [ $ARCH = "x86_64" ]; then
    ARCH="amd64"
fi

# echo "Installing zvm-$OS-$ARCH"

install_latest() {
    echo -e "Installing $1 in $(pwd)/zvm"
    if [ "$(uname)" = "Darwin" ]; then
     # Do something under MacOS platform

        if command -v wget >/dev/null 2>&1; then
    
            echo "wget is installed. Using wget..."
            wget -q --show-progress --max-redirect 5 -O zvm.tar "https://github.com/tristanisham/zvm/releases/latest/download/$1"
            tar -xf zvm.tar
            rm "zvm.tar"
        else
            echo "wget is not installed. Using curl..."
            curl "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.tar
        fi
        
    elif [ $OS = "Linux" ]; then
     # Do something under GNU/Linux platform
        if command -v wget >/dev/null 2>&1; then
    
            echo "wget is installed. Using wget..."
            wget -q --show-progress --max-redirect 5 -O zvm.tar "https://github.com/tristanisham/zvm/releases/latest/download/$1"
            tar -xf zvm.tar
            rm "zvm.tar"
        else
            echo "wget is not installed. Using curl..."
            curl "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.tar
        fi
    elif [ $OS = "MINGW32_NT" ]; then
    # Do something under 32 bits Windows NT platform
        curl "https://github.com/tristanisham/zvm/releases/latest/download/$($1) -o zvm.zip"

    elif [ $OS == "MINGW64_NT" ]; then
    # Do something under 64 bits Windows NT platform
        curl "https://github.com/tristanisham/zvm/releases/latest/download/$($1) -o zvm.zip"

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
echo "If this is your first time installing 'zvm' append the following to $HOME/.profile or $HOME/.bashrc"
echo -e "\x1b[1;32mexport PATH=\$PATH:\$HOME/.zvm/bin\x1b[1;0m"