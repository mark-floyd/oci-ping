#!/bin/bash

# Get OS and Architecture
OS=$(uname -s)
ARCH=$(uname -m)

BINARY=""
DOWNLOAD_URL=""

if [ "$OS" = "Darwin" ]; then
    BINARY="oci-ping-cli-mac-arm"
    DOWNLOAD_URL="https://github.com/mark-floyd/oci-ping/releases/latest/download/oci-ping-cli-mac-arm"
elif [ "$OS" = "Linux" ]; then
    if [ "$ARCH" = "x86_64" ]; then
        BINARY="oci-ping-cli-linux-x64"
        DOWNLOAD_URL="https://github.com/mark-floyd/oci-ping/releases/latest/download/oci-ping-cli-linux-x64"
    else
        echo "Error: Only x86_64 is supported for Linux at this time."
        exit 1
    fi
else
    echo "Error: Unsupported OS: $OS"
    exit 1
fi

# Download binary if it doesn't exist
if [ ! -f "$BINARY" ]; then
    echo "Binary $BINARY not found. Downloading from $DOWNLOAD_URL..."
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$BINARY" "$DOWNLOAD_URL"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$BINARY" "$DOWNLOAD_URL"
    else
        echo "Error: Neither curl nor wget found. Please install one to download the binary."
        exit 1
    fi
    chmod +x "$BINARY"
fi

# Execute the binary with all passed arguments
./"$BINARY" "$@"
