#!/bin/bash

set -e

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize architecture names
case $ARCH in
  x86_64)
    ARCH="amd64"
    ;;
  aarch64)
    ARCH="arm64"
    ;;
  armv7l)
    ARCH="arm"
    ;;
esac

# Construct binary name
BINARY_NAME="pr-check-notifier-${OS}-${ARCH}"

# Add .exe extension for Windows
if [[ $OS == "windows"* ]]; then
  BINARY_NAME="${BINARY_NAME}.exe"
fi

BINARY_PATH="${GITHUB_ACTION_PATH}/bin/${BINARY_NAME}"

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
  echo "::error::Binary not found for platform ${OS}-${ARCH}: ${BINARY_PATH}"
  echo "::error::Available binaries:"
  ls -la ${GITHUB_ACTION_PATH}/bin/pr-check-notifier-*
  exit 1
fi

# Make binary executable (in case git didn't preserve permissions)
chmod +x "$BINARY_PATH"

# Execute the binary
exec "$BINARY_PATH"