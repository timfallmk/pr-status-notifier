#!/bin/bash
set -e

# Build for common platforms
PLATFORMS=(
  "linux-amd64"
  "linux-arm64"
  "linux-arm"
  "darwin-amd64"
  "darwin-arm64"
  "windows-amd64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
  OS="${PLATFORM%-*}"
  ARCH="${PLATFORM#*-}"
  
  echo "Building for $OS-$ARCH..."
  if [ "$OS" = "windows" ]; then
    GOOS=$OS GOARCH=$ARCH go build -o "bin/pr-check-notifier-$OS-$ARCH.exe"
  else
    GOOS=$OS GOARCH=$ARCH go build -o "bin/pr-check-notifier-$OS-$ARCH"
  fi
done

# Make all binaries executable
chmod +x bin/pr-check-notifier-*

echo "Build complete! Created binaries:"
ls -la bin/pr-check-notifier-*