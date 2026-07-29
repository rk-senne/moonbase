#!/bin/sh
# moonbase installer — downloads the latest release binary for your platform,
# verifies its checksum, installs it, and sets up the agents.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/rk-senne/moonbase/main/install.sh | sh
#
# Environment overrides:
#   MOONBASE_REPO         GitHub owner/repo         (default: rk-senne/moonbase)
#   MOONBASE_INSTALL_DIR  install directory         (default: $HOME/.local/bin)
#   MOONBASE_VERSION      release tag, e.g. v1.6.0  (default: latest)
#
# Supports macOS and Linux on amd64/arm64.
set -eu

REPO="${MOONBASE_REPO:-rk-senne/moonbase}"
INSTALL_DIR="${MOONBASE_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${MOONBASE_VERSION:-latest}"

info() { printf '\033[36m•\033[0m %s\n' "$1"; }
ok()   { printf '\033[32m✓\033[0m %s\n' "$1"; }
die()  { printf '\033[31m✗ %s\033[0m\n' "$1" >&2; exit 1; }

# --- required tools ---
command -v curl >/dev/null 2>&1 || die "curl is required"
command -v tar  >/dev/null 2>&1 || die "tar is required"

# --- detect platform ---
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
	darwin|linux) ;;
	*) die "unsupported OS: $os (moonbase supports macOS and Linux)" ;;
esac

arch="$(uname -m)"
case "$arch" in
	x86_64|amd64) arch="amd64" ;;
	arm64|aarch64) arch="arm64" ;;
	*) die "unsupported architecture: $arch" ;;
esac

asset="moonbase_${os}_${arch}.tar.gz"

if [ "$VERSION" = "latest" ]; then
	base="https://github.com/${REPO}/releases/latest/download"
else
	base="https://github.com/${REPO}/releases/download/${VERSION}"
fi

# --- sha256 tool (optional but preferred) ---
if command -v sha256sum >/dev/null 2>&1; then
	sha_cmd="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
	sha_cmd="shasum -a 256"
else
	sha_cmd=""
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

info "Downloading ${asset} (${VERSION}) from ${REPO}..."
curl -fsSL "${base}/${asset}" -o "${tmp}/${asset}" \
	|| die "download failed — is the release public and does ${asset} exist?"

# --- verify checksum when available ---
if [ -n "$sha_cmd" ] && curl -fsSL "${base}/checksums.txt" -o "${tmp}/checksums.txt" 2>/dev/null; then
	expected="$(grep " ${asset}$" "${tmp}/checksums.txt" 2>/dev/null | awk '{print $1}')"
	if [ -n "$expected" ]; then
		actual="$(cd "$tmp" && $sha_cmd "$asset" | awk '{print $1}')"
		[ "$expected" = "$actual" ] || die "checksum mismatch for ${asset} (expected $expected, got $actual)"
		ok "checksum verified"
	fi
else
	info "checksums unavailable — skipping verification"
fi

# --- extract + install ---
tar -xzf "${tmp}/${asset}" -C "$tmp"
[ -f "${tmp}/moonbase" ] || die "archive did not contain a 'moonbase' binary"
mkdir -p "$INSTALL_DIR"
if ! install -m 0755 "${tmp}/moonbase" "${INSTALL_DIR}/moonbase" 2>/dev/null; then
	cp "${tmp}/moonbase" "${INSTALL_DIR}/moonbase"
	chmod 0755 "${INSTALL_DIR}/moonbase"
fi
ok "installed to ${INSTALL_DIR}/moonbase"

# --- install agents (embedded in the binary → ~/.moonbase/agents) ---
if "${INSTALL_DIR}/moonbase" setup >/dev/null 2>&1; then
	ok "agents installed to ~/.moonbase/agents"
else
	info "run 'moonbase setup' to install agents"
fi

# --- PATH hint ---
case ":$PATH:" in
	*":${INSTALL_DIR}:"*) ;;
	*) info "add ${INSTALL_DIR} to your PATH:  export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
esac

ok "moonbase ready — run:  moonbase        (or: moonbase init  in any project)"
