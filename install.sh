#!/usr/bin/env bash
# Install a2h from GitHub Releases.
#   curl -fsSL https://raw.githubusercontent.com/TyrellD1/agenttohuman/main/install.sh | bash
set -euo pipefail

REPO="${A2H_REPO:-TyrellD1/agenttohuman}"
INSTALL_DIR="${A2H_INSTALL_DIR:-$HOME/.local/bin}"
BASE="https://github.com/${REPO}/releases/latest/download"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac
case "$os" in
  darwin)
    if [ "$arch" != "arm64" ]; then
      echo "macOS is only supported on Apple Silicon (arm64)" >&2
      exit 1
    fi
    ;;
  linux) ;;
  *)
    echo "unsupported OS: $os (need macOS arm64 or Linux amd64/arm64)" >&2
    exit 1
    ;;
esac

asset="a2h-${os}-${arch}"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

tmp="$(mktemp -d)"
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT

echo "Downloading ${asset} from ${REPO} GitHub Releases..."
if ! curl -fsSL -A "a2h-install" -o "$tmp/$asset" "${BASE}/${asset}"; then
  echo "Download failed. Publish a release that includes ${asset}." >&2
  exit 1
fi
if ! curl -fsSL -A "a2h-install" -o "$tmp/SHA256SUMS" "${BASE}/SHA256SUMS"; then
  echo "Download failed. Release is missing SHA256SUMS." >&2
  exit 1
fi

expected="$(awk -v name="$asset" '$2 == name || $2 == "*"name { print $1; exit }' "$tmp/SHA256SUMS")"
if [ -z "$expected" ]; then
  echo "SHA256SUMS has no entry for ${asset}" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$asset" | awk '{print $1}')"
else
  actual="$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')"
fi
if [ "$actual" != "$expected" ]; then
  echo "checksum mismatch for ${asset}" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmp/$asset" "$INSTALL_DIR/a2h"

echo "installed ${INSTALL_DIR}/a2h"
if [ -x "${INSTALL_DIR}/a2h" ]; then
  echo "version: $("${INSTALL_DIR}/a2h" version)"
fi

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    marker="# added by a2h install"
    path_line="export PATH=\"${INSTALL_DIR}:\$PATH\""
    added=""
    for rc in "$HOME/.zshrc" "$HOME/.bashrc"; do
      if [ -f "$rc" ] && ! grep -q "$marker" "$rc" 2>/dev/null; then
        printf '\n%s\n%s\n' "$marker" "$path_line" >>"$rc"
        added="$rc"
        break
      fi
    done
    echo "${INSTALL_DIR} is not on PATH."
    if [ -n "$added" ]; then
      echo "Added it to ${added}. Open a new terminal, or run: source ${added}"
    else
      echo "Add this to your shell rc:"
      echo "  ${path_line}"
    fi
    ;;
esac
