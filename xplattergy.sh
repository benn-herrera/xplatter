#!/usr/bin/env bash
# xplattergy wrapper — resolves the correct binary for dev builds or dist packages.
set -e

SCRIPT_DIR=$(dirname "$0")
SCRIPT_DIR=$(cd "${SCRIPT_DIR}" && pwd)

# Dev build: prefer bin/xplattergy (built by `make build`)
if [ -x "$SCRIPT_DIR/bin/xplattergy" ]; then
    exec "$SCRIPT_DIR/bin/xplattergy" "$@"
fi

# Dist package: detect OS and ARCH, find bin/xplattergy-OS-ARCH
case "$(uname -s)" in
    Darwin)  OS=darwin  ;;
    Linux)   OS=linux   ;;
    MINGW*|MSYS*|CYGWIN*) OS=windows ;;
    *) echo "xplattergy: unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
    x86_64)        ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
    *) echo "xplattergy: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

BIN="$SCRIPT_DIR/bin/xplattergy-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    BIN="${BIN}.exe"
fi

if [ -x "$BIN" ]; then
    exec "$BIN" "$@"
fi

echo "xplattergy: no binary found." >&2
echo "  If you've downloaded a dist package, run build_codegen.sh to build from source." >&2
echo "  If you're a developer 'make build'" >&2
exit 1
