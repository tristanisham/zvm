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
        else
            echo "wget is not installed. Using curl..."
            curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$1" -o zvm.tar
        fi
        
        mkdir -p $HOME/.zvm/self
        tar -xf zvm.tar -C $HOME/.zvm/self
        rm "zvm.tar"
        
    elif [ $OS = "Linux" ]; then
     # Do something under GNU/Linux platform
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
    elif [ $OS = "MINGW32_NT" ]; then
    # Do something under 32 bits Windows NT platform
        curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$($1)" -o zvm.zip

    elif [ $OS == "MINGW64_NT" ]; then
    # Do something under 64 bits Windows NT platform
        curl -L --max-redirs 5 "https://github.com/tristanisham/zvm/releases/latest/download/$($1)" -o zvm.zip

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

# Check if ZVM_INSTALL is not set
if [ -z "$ZVM_INSTALL" ]; then
    # Append the lines to $HOME/.profile
    echo 'export ZVM_INSTALL="$HOME/.zvm/self"' >> $HOME/.profile
    echo 'export PATH="$PATH:/home/tristan/.zvm/bin"' >> $HOME/.profile
    echo 'export PATH="$ZVM_INSTALL/self:$PATH"' >> $HOME/.profile
fi

echo "Run 'source ~/.profile' to start using ZVM in this shell!"