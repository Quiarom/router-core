#!/usr/bin/env bash
set -euo pipefail

# Compila los servicios Go con el nombre que Tauri exige para la plataforma actual.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if ! command -v go >/dev/null 2>&1 && [ -x "$HOME/.local/go/bin/go" ]; then
  export PATH="$HOME/.local/go/bin:$PATH"
fi
if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go no está instalado o no está en el PATH" >&2
  exit 1
fi
TARGET_TRIPLE="$(rustc --print host-tuple)"
OUTPUT_DIR="$ROOT_DIR/frontend/src-tauri/binaries"

case "$TARGET_TRIPLE" in
  x86_64-unknown-linux-gnu)
    GO_ARCH="amd64"
    ;;
  aarch64-unknown-linux-gnu)
    GO_ARCH="arm64"
    ;;
  *)
    echo "Error: Fedora no compatible con el objetivo Rust $TARGET_TRIPLE" >&2
    exit 1
    ;;
esac

mkdir -p "$OUTPUT_DIR"
cd "$ROOT_DIR"

CGO_ENABLED=0 GOOS=linux GOARCH="$GO_ARCH" go build \
  -o "$OUTPUT_DIR/router-core-$TARGET_TRIPLE" \
  ./cmd/router-core
CGO_ENABLED=0 GOOS=linux GOARCH="$GO_ARCH" go build \
  -o "$OUTPUT_DIR/router-core-agent-$TARGET_TRIPLE" \
  ./cmd/router-core-agent

chmod 0755 \
  "$OUTPUT_DIR/router-core-$TARGET_TRIPLE" \
  "$OUTPUT_DIR/router-core-agent-$TARGET_TRIPLE"

echo "Sidecars preparados para $TARGET_TRIPLE"
