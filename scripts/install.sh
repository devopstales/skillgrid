#!/usr/bin/env bash
# Install aiskillgrid from GitHub Releases onto PATH (macOS / Linux).
set -euo pipefail

REPO="${AISKILLGRID_REPO:-aiskillgrid/aiskillgrid}"
VERSION="${AISKILLGRID_VERSION:-latest}"
INSTALL_DIR="${AISKILLGRID_INSTALL_DIR:-${HOME}/.local/bin}"
NAME="aiskillgrid"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "${arch}" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "unsupported arch: ${arch}" >&2; exit 1 ;;
esac
case "${os}" in
  darwin|linux) ;;
  *) echo "unsupported OS: ${os} (use install.ps1 on Windows)" >&2; exit 1 ;;
esac

asset="${NAME}-${VERSION}-${os}-${arch}"
if [[ "${VERSION}" == "latest" ]]; then
  # Resolve latest tag via API, then map asset name pattern NAME-VERSION-OS-ARCH
  tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)"
  if [[ -z "${tag}" ]]; then
    echo "could not resolve latest release for ${REPO}" >&2
    exit 1
  fi
  VERSION="${tag#v}"
  asset="${NAME}-${VERSION}-${os}-${arch}"
fi

url="https://github.com/${REPO}/releases/download/v${VERSION}/${asset}"
# also try without v prefix on tag
tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
echo "Downloading ${url}"
if ! curl -fsSL -o "${tmpdir}/${NAME}" "${url}"; then
  url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"
  echo "Retrying ${url}"
  curl -fsSL -o "${tmpdir}/${NAME}" "${url}"
fi
chmod +x "${tmpdir}/${NAME}"
mkdir -p "${INSTALL_DIR}"
mv "${tmpdir}/${NAME}" "${INSTALL_DIR}/${NAME}"
echo "Installed ${INSTALL_DIR}/${NAME}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "Add to PATH: export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
esac
echo "Next: aiskillgrid sync && aiskillgrid install"
