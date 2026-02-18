#!/usr/bin/env bash
# xplatter wrapper — resolves the correct binary for dev builds or dist packages.
set -e

SCRIPT_DIR=$(dirname "$0")
SCRIPT_DIR=$(cd "${SCRIPT_DIR}" && pwd)

OS=$(uname -s)
ARCH=$(uname -m)
EXE=
case "$OS" in
    Darwin)  OS=darwin  ;;
    Linux)   OS=linux   ;;
    MINGW*|MSYS*|CYGWIN*) OS=windows; EXE=.exe ;;
    *) echo "xplatter: unsupported OS: $OS" >&2; exit 1 ;;
esac
case "$ARCH" in
    x86_64)        ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
    *) echo "xplatter: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

is_valid() {
    local bin="$1"
    # silently run version to get an exit code.
    # if there's some weird os version incompatibility
    # with the prebuilt executable this will catch it.
    [[ -x "$bin" ]] && ("$bin" version 1>&2) 2> /dev/null
}

BIN="$SCRIPT_DIR/bin/xplatter${EXE}"
# Dev build: prefer bin/xplatter (built by `make build`)
if ! is_valid "$BIN"; then
    # Dist package: use OS and ARCH to construct name xplatter-OS-ARCH
    BIN="$SCRIPT_DIR/bin/xplatter-${OS}-${ARCH}${EXE}"    
fi

if ! is_valid "$BIN"; then
    echo "xplatter: no usable binary found." >&2
    if [[ -d "${SCRIPT_DIR}"/.git ]]; then
        echo "  'make build' first." >&2        
    else
        echo "  'build_codegen.sh' to build from source." >&2        
    fi
    exit 1
fi

exec "$BIN" "$@"

