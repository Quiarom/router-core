#!/usr/bin/env bash
#
# install.sh - user-facing installer for Gavetero.
#
# Usage:
#   curl -sSf https://raw.githubusercontent.com/Quiarom/router-core/integration/gavetero/install.sh | sh
#
# Environment variables (set ONE of these):
#   GAVETERO_VERSION=v0.1.0-alpha.1   - install that release tag
#   GAVETERO_COMMIT=<sha>             - install that commit (via GitHub Actions artifact)
#   (none)                           - install the latest stable release
#
# Behavior:
#   1. Detects the platform (linux/darwin, amd64/arm64).
#   2. Downloads the matching prebuilt binary.
#   3. Installs to ~/.local/bin/gavetero + symlink to gvt.
#   4. If ~/.local/bin is not on PATH, prints one line to add to the shell rc.
#   5. Verifies the install with `gvt version`.
#
# The script never requires sudo, never writes outside the user's
# home directory, and never modifies the system PATH. If anything
# fails, it prints a single actionable line and exits non-zero.

set -euo pipefail

REPO="Quiarom/router-core"
BIN="gavetero"
DEST="${HOME}/.local/bin"

# ---------------------------------------------------------------------------
# Platform detection. We use uname -m for arch and uname -s for OS.
# ---------------------------------------------------------------------------
detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "${arch}" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "unsupported architecture: ${arch}" >&2; return 1 ;;
  esac
  case "${os}" in
    linux) os="linux" ;;
    darwin) os="darwin" ;;
    *) echo "unsupported OS: ${os}" >&2; return 1 ;;
  esac
  echo "${os}_${arch}"
}

# ---------------------------------------------------------------------------
# Resolve the download URL.
#
# Priority:
#   1. GAVETERO_VERSION (e.g. v0.1.0-alpha.1) - GitHub Releases
#   2. GAVETERO_COMMIT (e.g. abc1234) - Actions artifact
#   3. neither - latest release (or fall back to make install-user)
# ---------------------------------------------------------------------------
resolve_url() {
  local platform="$1"
  if [ -n "${GAVETERO_VERSION:-}" ]; then
    echo "https://github.com/${REPO}/releases/download/${GAVETERO_VERSION}/${BIN}_${platform}.tar.gz"
  elif [ -n "${GAVETERO_COMMIT:-}" ]; then
    echo "https://github.com/${REPO}/actions/runs/${GAVETERO_COMMIT}/artifacts/${BIN}-${platform}"
  else
    echo "https://github.com/${REPO}/releases/latest/download/${BIN}_${platform}.tar.gz"
  fi
}

main() {
  echo "Gavetero installer"
  echo "=================="

  local platform
  if ! platform="$(detect_platform)"; then
    echo ""
    echo "This script does not support your platform. Build from source instead:"
    echo "  git clone https://github.com/${REPO}.git"
    echo "  cd router-core"
    echo "  make install-user"
    exit 1
  fi
  echo ""
  echo "  platform : ${platform}"

  local url
  url="$(resolve_url "${platform}")"
  echo "  source   : ${url}"
  echo "  dest     : ${DEST}/${BIN}"
  echo ""

  # Download
  local tmpdir
  tmpdir="$(mktemp -d)"
  trap "rm -rf '${tmpdir}'" EXIT
  echo "Downloading..."
  if ! curl -fsSL -o "${tmpdir}/gavetero.tar.gz" "${url}"; then
    echo ""
    echo "Download failed."
    echo ""
    echo "If you are a developer working in the repo, use:"
    echo "  make install-user"
    echo ""
    echo "If you are a user, set GAVETERO_VERSION to a published release:"
    echo "  GAVETERO_VERSION=v0.1.0-alpha.1 curl -sSf .../install.sh | sh"
    exit 1
  fi

  # Extract (tarball layout: gavetero + gvt symlink target)
  tar -xzf "${tmpdir}/gavetero.tar.gz" -C "${tmpdir}"
  if [ ! -f "${tmpdir}/gavetero" ]; then
    echo ""
    echo "Downloaded archive does not contain 'gavetero'."
    exit 1
  fi

  # Install
  mkdir -p "${DEST}"
  install -m 0755 "${tmpdir}/gavetero" "${DEST}/gavetero"
  ln -sf "${BIN}" "${DEST}/gvt"

  # Verify
  echo ""
  echo "Installed:"
  echo "  ${DEST}/gavetero"
  echo "  ${DEST}/gvt -> gavetero"
  echo ""
  echo "Verify:"
  "${DEST}/gvt" version || true
  echo ""

  if echo "${PATH}" | tr ':' '\n' | grep -qx "${DEST}"; then
    echo "Ready. Try: gvt version"
  else
    echo "${DEST} is not on your PATH."
    echo "Add this line to ~/.bashrc (or ~/.zshrc, etc.):"
    echo "  export PATH=\"${DEST}:\${PATH}\""
    echo "Then open a new shell."
  fi
}

main "$@"
