#!/bin/bash

# Get OS and Architecture
OS=$(uname -s)
ARCH=$(uname -m)

BINARY=""

if [ "$OS" = "Darwin" ]; then
    if [ "$ARCH" = "arm64" ]; then
        BINARY="./oci-ping-cli-mac-arm"
    else
        echo "Error: Mac Intel binary not found. Please build it for your architecture."
        exit 1
    fi
elif [ "$OS" = "Linux" ]; then
    if [ "$ARCH" = "x86_64" ]; then
        BINARY="./oci-ping-cli-linux-x64"
    else
        echo "Error: Linux ARM binary not found. Please build it for your architecture."
        exit 1
    fi
else
    echo "Error: Unsupported OS: $OS"
    exit 1
fi

if [ ! -f "$BINARY" ]; then
    echo "Error: Binary $BINARY not found. Building it now..."
    cd oci-ping-cli && go build -o "../$BINARY" && cd ..
fi

# Execute the binary with all passed arguments
$BINARY "$@"
