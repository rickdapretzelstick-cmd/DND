#!/usr/bin/env bash
# Builds the Windows app (and optional Mac/Linux builds) into dist/.
# Usage: tools/build.sh [version]     e.g. tools/build.sh 0.2.0
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${1:-0.1.0}"
mkdir -p dist

# Windows icon + version info, embedded via .syso resource files.
GOFLAGS=-mod=mod go run github.com/tc-hib/go-winres@v0.3.3 simply \
  --icon winres/icon.png --manifest cli \
  --product-name "D&D Campaign Runner" --file-description "D&D Campaign Runner" \
  --product-version "$VERSION" --file-version "$VERSION" \
  --arch amd64,arm64

LDFLAGS="-s -w -X main.version=$VERSION"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o "dist/DnD-Campaign-Runner.exe" .
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "$LDFLAGS" -o "dist/DnD-Campaign-Runner-arm64.exe" .
if [[ "${ALL:-}" == "1" ]]; then
  CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$LDFLAGS" -o "dist/dnd-campaign-runner-mac" .
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o "dist/dnd-campaign-runner-linux" .
fi
ls -lh dist
