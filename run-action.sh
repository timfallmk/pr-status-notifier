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

# Debug: Print environment variables
echo "Checking environment variables:"
echo "GITHUB_TOKEN: ${GITHUB_TOKEN:-(not set)}"
echo "INPUT_GITHUB_TOKEN: ${INPUT_GITHUB_TOKEN:-(not set)}"

# Execute the binary with explicit environment variables
exec env \
  GITHUB_TOKEN="$GITHUB_TOKEN" \
  INPUT_GITHUB_TOKEN="$INPUT_GITHUB_TOKEN" \
  INPUT_EXCLUDED_CHECKS="$INPUT_EXCLUDED_CHECKS" \
  INPUT_NOTIFICATION_MESSAGE="$INPUT_NOTIFICATION_MESSAGE" \
  "$BINARY_PATH"