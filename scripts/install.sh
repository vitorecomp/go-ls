#!/usr/bin/env bash
# Build and install go-ls, go-ls-server, go-ls-client, and the compiled Next.js
# Web UI bundle globally on the system or into the current user's PATH.
#
# Usage:
#   ./scripts/install.sh                  # Auto-selects writable global bin dir on PATH (or /usr/local if root)
#   ./scripts/install.sh --system         # Install system-wide into /usr/local/bin and /usr/local/share/go-ls/web (uses sudo if needed)
#   ./scripts/install.sh --prefix=/path   # Install into <prefix>/bin and <prefix>/share/go-ls/web
#   ./scripts/install.sh --skip-web       # Build and install Go binaries only (skip Next.js build)
#   ./scripts/install.sh --uninstall      # Remove globally installed go-ls binaries and web UI assets

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

ACTION="install"
SYSTEM_WIDE=false
SKIP_WEB=false
CUSTOM_PREFIX="${PREFIX:-}"
CUSTOM_BINDIR="${BINDIR:-}"
CUSTOM_DATADIR="${DATADIR:-}"

usage() {
  cat <<EOF
Usage: ./scripts/install.sh [options]

Options:
  --system            Install system-wide to /usr/local (uses sudo if not root)
  --prefix=<dir>      Installation prefix (default: /usr/local when root, or ~/.local)
  --bin-dir=<dir>     Override binary installation directory
  --data-dir=<dir>    Override static web UI asset directory
  --skip-web          Skip building and installing the Next.js static frontend
  --uninstall         Uninstall go-ls binaries and static web UI assets
  -h, --help          Show this help message
EOF
}

for arg in "$@"; do
  case "$arg" in
    --system)
      SYSTEM_WIDE=true
      ;;
    --prefix=*)
      CUSTOM_PREFIX="${arg#*=}"
      ;;
    --bin-dir=*)
      CUSTOM_BINDIR="${arg#*=}"
      ;;
    --data-dir=*)
      CUSTOM_DATADIR="${arg#*=}"
      ;;
    --skip-web)
      SKIP_WEB=true
      ;;
    --uninstall|uninstall)
      ACTION="uninstall"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $arg" >&2
      usage >&2
      exit 1
      ;;
  esac
done

detect_default_dirs() {
  if [[ "$SYSTEM_WIDE" == "true" ]]; then
    TARGET_PREFIX="${CUSTOM_PREFIX:-/usr/local}"
    TARGET_BINDIR="${CUSTOM_BINDIR:-${TARGET_PREFIX}/bin}"
    TARGET_DATADIR="${CUSTOM_DATADIR:-${TARGET_PREFIX}/share/go-ls/web}"
    return
  fi

  if [[ -n "$CUSTOM_PREFIX" ]]; then
    TARGET_PREFIX="$CUSTOM_PREFIX"
    TARGET_BINDIR="${CUSTOM_BINDIR:-${TARGET_PREFIX}/bin}"
    TARGET_DATADIR="${CUSTOM_DATADIR:-${TARGET_PREFIX}/share/go-ls/web}"
    return
  fi

  if [[ "$(id -u)" -eq 0 ]] || [[ -w "/usr/local/bin" && -w "/usr/local/share" ]]; then
    TARGET_PREFIX="/usr/local"
    TARGET_BINDIR="${CUSTOM_BINDIR:-/usr/local/bin}"
    TARGET_DATADIR="${CUSTOM_DATADIR:-/usr/local/share/go-ls/web}"
    return
  fi

  TARGET_PREFIX="${HOME}/.local"
  TARGET_DATADIR="${CUSTOM_DATADIR:-${HOME}/.local/share/go-ls/web}"

  if [[ -n "$CUSTOM_BINDIR" ]]; then
    TARGET_BINDIR="$CUSTOM_BINDIR"
    return
  fi

  GOBIN_DIR="$(go env GOBIN 2>/dev/null || true)"
  if [[ -z "$GOBIN_DIR" ]]; then
    GOPATH_DIR="$(go env GOPATH 2>/dev/null || true)"
    if [[ -n "$GOPATH_DIR" ]]; then
      GOBIN_DIR="${GOPATH_DIR%%:*}/bin"
    fi
  fi

  if [[ ":${PATH}:" == *":${HOME}/.local/bin:"* ]]; then
    TARGET_BINDIR="${HOME}/.local/bin"
  elif [[ -n "$GOBIN_DIR" && ":${PATH}:" == *":${GOBIN_DIR}:"* ]]; then
    TARGET_BINDIR="$GOBIN_DIR"
  else
    TARGET_BINDIR="${HOME}/.local/bin"
  fi
}

run_priv() {
  local check_dir="$1"
  shift
  while [[ ! -e "$check_dir" && "$check_dir" != "/" && "$check_dir" != "." ]]; do
    check_dir="$(dirname "$check_dir")"
  done
  if [[ -w "$check_dir" ]]; then
    "$@"
  elif command -v sudo >/dev/null 2>&1; then
    sudo "$@"
  else
    "$@"
  fi
}

detect_default_dirs

if [[ "$ACTION" == "uninstall" ]]; then
  echo "Uninstalling go-ls..."
  CANDIDATE_BINDIRS=("$TARGET_BINDIR")
  if [[ -z "$CUSTOM_PREFIX" && -z "$CUSTOM_BINDIR" && "$SYSTEM_WIDE" == "false" ]]; then
    GOBIN_FALLBACK="$(go env GOBIN 2>/dev/null || echo "$(go env GOPATH 2>/dev/null)/bin")"
    CANDIDATE_BINDIRS+=("${HOME}/.local/bin" "$GOBIN_FALLBACK" "/usr/local/bin")
  fi

  REMOVED_ANY=false
  for bdir in "${CANDIDATE_BINDIRS[@]}"; do
    [[ -z "$bdir" ]] && continue
    for bin in go-ls go-ls-server go-ls-client; do
      if [[ -f "${bdir}/${bin}" ]]; then
        if [[ -w "$bdir" ]]; then
          rm -f "${bdir}/${bin}"
          echo "  ✓ Removed ${bdir}/${bin}"
          REMOVED_ANY=true
        elif [[ "$SYSTEM_WIDE" == "true" ]] && command -v sudo >/dev/null 2>&1; then
          sudo rm -f "${bdir}/${bin}"
          echo "  ✓ Removed ${bdir}/${bin}"
          REMOVED_ANY=true
        fi
      fi
    done
  done

  CANDIDATE_DATADIRS=("$TARGET_DATADIR" "${HOME}/.local/share/go-ls")
  if [[ "$SYSTEM_WIDE" == "true" || "$(id -u)" -eq 0 ]]; then
    CANDIDATE_DATADIRS+=("/usr/local/share/go-ls")
  fi
  for ddir in "${CANDIDATE_DATADIRS[@]}"; do
    if [[ -d "$ddir" ]]; then
      if [[ -w "$ddir" || -w "$(dirname "$ddir")" ]]; then
        rm -rf "$ddir"
        echo "  ✓ Removed ${ddir}"
        REMOVED_ANY=true
      elif [[ "$SYSTEM_WIDE" == "true" ]] && command -v sudo >/dev/null 2>&1; then
        sudo rm -rf "$ddir"
        echo "  ✓ Removed ${ddir}"
        REMOVED_ANY=true
      fi
    fi
  done

  if [[ "$REMOVED_ANY" == "false" ]]; then
    echo "  No installed go-ls binaries or assets found."
  else
    echo "Uninstall complete."
  fi
  exit 0
fi

echo "==> Building go-ls binaries..."
TMP_BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_BUILD_DIR"' EXIT

go build -ldflags="-s -w" -o "${TMP_BUILD_DIR}/go-ls" .
go build -ldflags="-s -w" -o "${TMP_BUILD_DIR}/go-ls-server" ./cmd/server
go build -ldflags="-s -w" -o "${TMP_BUILD_DIR}/go-ls-client" ./cmd/client

echo "==> Installing binaries to ${TARGET_BINDIR}..."
run_priv "$TARGET_BINDIR" mkdir -p "$TARGET_BINDIR"
for bin in go-ls go-ls-server go-ls-client; do
  run_priv "$TARGET_BINDIR" install -m 0755 "${TMP_BUILD_DIR}/${bin}" "${TARGET_BINDIR}/${bin}"
  echo "  ✓ Installed ${TARGET_BINDIR}/${bin}"
done

if [[ "$SKIP_WEB" == "false" ]]; then
  if [[ ! -f "web/out/index.html" ]]; then
    if command -v npm >/dev/null 2>&1; then
      echo "==> Building Next.js static frontend (web/out)..."
      (cd web && npm install --no-audit --no-fund && npm run build)
    else
      echo "  ! npm not found; skipping Next.js static frontend build."
    fi
  fi

  if [[ -f "web/out/index.html" ]]; then
    echo "==> Installing Next.js static frontend to ${TARGET_DATADIR}..."
    run_priv "$TARGET_DATADIR" mkdir -p "$TARGET_DATADIR"
    run_priv "$TARGET_DATADIR" cp -R web/out/. "$TARGET_DATADIR/"
    echo "  ✓ Installed Web UI assets to ${TARGET_DATADIR}"
  fi
fi

echo ""
echo "go-ls installed successfully!"
echo "  Binaries : ${TARGET_BINDIR}/{go-ls,go-ls-server,go-ls-client}"
if [[ -d "$TARGET_DATADIR" ]]; then
  echo "  Web UI   : ${TARGET_DATADIR}"
fi

if [[ ":${PATH}:" != *":${TARGET_BINDIR}:"* ]]; then
  echo ""
  echo "NOTE: ${TARGET_BINDIR} is not currently in your PATH."
  echo "Add the following line to your shell profile (~/.bashrc or ~/.zshrc):"
  echo "  export PATH=\"${TARGET_BINDIR}:\$PATH\""
fi
