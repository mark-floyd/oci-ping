#!/bin/bash

# Get OS and Architecture
OS=$(uname -s)
ARCH=$(uname -m)

BINARY=""
DOWNLOAD_URL=""

if [ "$OS" = "Darwin" ]; then
    if [ "$ARCH" = "arm64" ]; then
        BINARY="oci-ping-cli-darwin-arm64"
        DOWNLOAD_URL="https://ghfast.top/https://github.com/mark-floyd/oci-ping/releases/latest/download/oci-ping-cli-darwin-arm64"
    else
        echo "Error: macOS on x86_64 (Intel) is not supported at this time."
        exit 1
    fi
elif [ "$OS" = "Linux" ]; then
    if [ "$ARCH" = "x86_64" ]; then
        BINARY="oci-ping-cli-linux-x64"
        DOWNLOAD_URL="https://ghfast.top/https://github.com/mark-floyd/oci-ping/releases/latest/download/oci-ping-cli-linux-x64"
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
        BINARY="oci-ping-cli-linux-arm64"
        DOWNLOAD_URL="https://ghfast.top/https://github.com/mark-floyd/oci-ping/releases/latest/download/oci-ping-cli-linux-arm64"
    else
        echo "Error: Unsupported architecture for Linux: $ARCH"
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
        curl -L -f -s -o "$BINARY" "$DOWNLOAD_URL"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$BINARY" "$DOWNLOAD_URL"
    else
        echo "Error: Neither curl nor wget found."
        exit 1
    fi
    
    if [ $? -ne 0 ]; then
        echo "Error: Download failed. The release might not exist or the asset name is incorrect."
        rm -f "$BINARY"
        exit 1
    fi

    # More robust binary check: Check for ELF or Mach-O magic numbers
    # Linux ELF: \x7fELF
    # Mac Mach-O: \xcf\xfa\xed\xfe or \xce\xfa\xed\xfe
    MAGIC=$(head -c 4 "$BINARY" | xxd -p 2>/dev/null || od -t x1 -N 4 "$BINARY" | head -n 1 | cut -d' ' -f2-5 | tr -d ' ' 2>/dev/null)
    
    if [[ "$MAGIC" != "7f454c46" && "$MAGIC" != "cffaedfe" && "$MAGIC" != "cefaedfe" && "$MAGIC" != "feedface" && "$MAGIC" != "feedfacf" ]]; then
        echo "Error: Downloaded file is not a valid binary (Magic: $MAGIC). It might be an error page."
        rm -f "$BINARY"
        exit 1
    fi

    chmod +x "$BINARY"
    echo "Download successful."
fi

# Execute the binary with all passed arguments
./"$BINARY" "$@"

# Cleanup: delete the binary after running
if [ -f "$BINARY" ]; then
    rm -f "$BINARY"
fi
