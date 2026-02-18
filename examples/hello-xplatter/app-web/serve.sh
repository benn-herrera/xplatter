#!/usr/bin/env bash
set -euo pipefail

PORT="${1:-8080}"
DIR="$(cd "$(dirname "$0")" && pwd)/build"

if [ ! -d "$DIR" ]; then
  echo "Error: build/ directory not found. Run 'make stage' first." >&2
  exit 1
fi

echo "Serving $DIR on http://localhost:$PORT"

if command -v python3 &>/dev/null; then
  python3 -m http.server "$PORT" --directory "$DIR"
elif command -v node &>/dev/null; then
  npx -y http-server "$DIR" -p "$PORT"
else
  echo "Error: neither python3 nor node found." >&2
  echo "Install Python 3: https://www.python.org/downloads/" >&2
  echo "Install Node.js:  https://nodejs.org/" >&2
  exit 1
fi
