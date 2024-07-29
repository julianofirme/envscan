#!/bin/bash

REPO="jfirme-sys/envscan"
LATEST_RELEASE=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep "tag_name" | awk '{print substr($2, 2, length($2)-3) }')
OS=$(uname -s)
ARCH=$(uname -m)

case $OS in
    Linux)
        BINARY="envscan-linux"
        ;;
    Darwin)
        BINARY="envscan-darwin"
        ;;
    CYGWIN*|MINGW32*|MSYS*|MINGW*)
        BINARY="envscan-windows.exe"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

curl -LO "https://github.com/$REPO/releases/download/$LATEST_RELEASE/$BINARY"
chmod +x $BINARY
sudo mv $BINARY /usr/local/bin/envscan

echo "envscan have been successfully instaled in /usr/local/bin"
