#!/usr/bin/env bash
set -euo pipefail

# build_codegen.sh — Build the xplattergy code generation tool.
#
# Detects or installs Go, then builds the binary.
#
# Usage:
#   ./build_codegen.sh              # build for the current platform
#   ./build_codegen.sh --version    # print the version that would be embedded
#   ./build_codegen.sh --help       # show usage
#
# Environment variables:
#   XPLATTERGY_VERSION  Override the embedded version string (default: git describe or "dev")
#   GO_MIN_VERSION      Minimum required Go version (default: 1.22)
#   GOBIN               Directory for the built binary (default: ./bin)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC_DIR="${SCRIPT_DIR}/src"
GO_MIN_VERSION="${GO_MIN_VERSION:-1.22}"
GOBIN="${GOBIN:-${SCRIPT_DIR}/bin}"
MODULE_PATH="github.com/benn-herrera/xplattergy"

# ---------- helpers ----------

die() { echo "error: $*" >&2; exit 1; }

log() { echo "==> $*"; }

# Compare two dotted version strings. Returns 0 if $1 >= $2.
version_gte() {
    local IFS=.
    local i a=($1) b=($2)
    for ((i = 0; i < ${#b[@]}; i++)); do
        local av="${a[i]:-0}"
        local bv="${b[i]:-0}"
        if ((av > bv)); then return 0; fi
        if ((av < bv)); then return 1; fi
    done
    return 0
}

# Detect the installed Go version. Prints the version string (e.g. "1.22.4")
# or returns 1 if Go is not found.
detect_go() {
    local go_bin="${1:-go}"
    if ! command -v "$go_bin" &>/dev/null; then
        return 1
    fi
    "$go_bin" version | sed -n 's/.*go\([0-9][0-9]*\.[0-9][0-9]*\(\.[0-9][0-9]*\)*\).*/\1/p' | head -1
}

# Resolve version string for embedding.
resolve_version() {
    if [[ -n "${XPLATTERGY_VERSION:-}" ]]; then
        echo "$XPLATTERGY_VERSION"
        return
    fi
    if git -C "$SCRIPT_DIR" describe --tags --always 2>/dev/null; then
        return
    fi
    echo "dev"
}

# ---------- Go detection / installation guidance ----------

ensure_go() {
    local go_ver
    if go_ver="$(detect_go)"; then
        if version_gte "$go_ver" "$GO_MIN_VERSION"; then
            log "Found Go ${go_ver} (>= ${GO_MIN_VERSION})"
            return 0
        fi
        die "Go ${go_ver} found but >= ${GO_MIN_VERSION} is required. Please upgrade: https://go.dev/dl/"
    fi

    echo ""
    echo "Go is not installed. Install it using one of:"
    echo ""
    echo "  macOS (Homebrew):  brew install go"
    echo "  Linux (apt):       sudo apt-get install golang"
    echo "  Linux (snap):      sudo snap install go --classic"
    echo "  Any platform:      https://go.dev/dl/"
    echo ""
    die "Go >= ${GO_MIN_VERSION} is required to build xplattergy."
}

# ---------- build ----------

do_build() {
    local version
    version="$(resolve_version)"

    log "Building xplattergy (version: ${version})"

    mkdir -p "$GOBIN"

    local ldflags="-s -w -X ${MODULE_PATH}/cmd.Version=${version}"

    (cd "$SRC_DIR" && go build \
        -ldflags "$ldflags" \
        -o "${GOBIN}/xplattergy" \
        .)

    log "Built: ${GOBIN}/xplattergy"
}

# ---------- main ----------

case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [--version | --help]"
        echo ""
        echo "Build the xplattergy code generation tool."
        echo ""
        echo "Options:"
        echo "  --version   Print the version that would be embedded and exit"
        echo "  --help      Show this help message"
        echo ""
        echo "Environment variables:"
        echo "  XPLATTERGY_VERSION  Override embedded version (default: git describe or 'dev')"
        echo "  GO_MIN_VERSION      Minimum Go version (default: 1.22)"
        echo "  GOBIN               Output directory (default: ./bin)"
        exit 0
        ;;
    --version)
        resolve_version
        exit 0
        ;;
    "")
        ensure_go
        do_build
        ;;
    *)
        die "Unknown argument: $1. Use --help for usage."
        ;;
esac
