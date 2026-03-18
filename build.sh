#!/usr/bin/env bash
# build.sh — local cross-compile helper
# Usage: ./build.sh [version]
#   version defaults to the output of `git describe --tags --always`

set -e

VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo 'dev')}"
LDFLAGS="-X github.com/boy-hack/ksubdomain/v2/pkg/core/conf.Version=${VERSION}"

echo "Building version: ${VERSION}"

# Linux amd64 (static libpcap required on the build host)
CGO_LDFLAGS="-Wl,-static -L/usr/lib/x86_64-linux-gnu/libpcap.a -lpcap -Wl,-Bdynamic -ldbus-1 -lsystemd" \
  GOOS=linux GOARCH=amd64 \
  go build -ldflags "${LDFLAGS}" -o ./ksubdomain_linux_amd64 ./cmd/ksubdomain/
echo "  -> ksubdomain_linux_amd64"

# Windows amd64 (CGO disabled; npcap is linked at runtime)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags "${LDFLAGS}" -o ./ksubdomain_windows_amd64.exe ./cmd/ksubdomain/
echo "  -> ksubdomain_windows_amd64.exe"

echo "Done."
