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
    # Do something under Mac OS X platform
        wget -q --show-progress --max-redirect 5 -O zvm.tar.gz "https://github.com/tristanisham/zvm/releases/latest/download/$1"
        tar -xf zvm.tar.gz
        rm "zvm.tar.gz"
    elif [ $OS = "Linux" ]; then
     # Do something under GNU/Linux platform
        wget -q --show-progress --max-redirect 5 -O zvm.tar.gz "https://github.com/tristanisham/zvm/releases/latest/download/$1"
        tar -xf zvm.tar.gz
        rm "zvm.tar.gz"
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
    install_latest "zvm-darwin-$ARCH.tar.gz"
elif [ $OS = "Linux" ]; then
     # Do something under GNU/Linux platform
    install_latest "zvm-linux-$ARCH.tar.gz"
elif [ $OS = "MINGW32_NT" ]; then
    # Do something under 32 bits Windows NT platform
    install_latest "zvm-windows-$ARCH.zip"
elif [ $OS == "MINGW64_NT" ]; then
    # Do something under 64 bits Windows NT platform
    install_latest "zvm-windows-$ARCH.zip"
fi

echo
echo "Append the following to your $HOME/.profile or $HOME/.bash_rc"
echo
echo -e "\texport PATH=\$PATH:\$HOME/.zvm/bin"
echo