#!/usr/bin/env bash
# Seed realistic multi-computer, multi-drive, and multi-revision fake data
# into a running go-ls server (defaults to http://localhost:8080).
#
# Usage:
#   ./scripts/seed-fake-data.sh
#   ./scripts/seed-fake-data.sh http://localhost:8080
#   ./scripts/seed-fake-data.sh --server=http://localhost:8080

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

SERVER_URL="${GOLS_SERVER_URL:-http://localhost:8080}"

if [[ $# -gt 0 ]]; then
  case "$1" in
    http://*|https://*)
      SERVER_URL="$1"
      shift
      ;;
  esac
fi

exec go run ./scripts/seed --server="$SERVER_URL" "$@"
