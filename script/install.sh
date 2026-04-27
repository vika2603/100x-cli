#!/usr/bin/env sh
# Install 100x-cli by downloading a release archive that matches the host
# OS/arch, verifying its SHA-256, and copying the binary into a directory
# on PATH. POSIX shell, no third-party deps. Linux and macOS only; on
# Windows, use `go install` or download an asset manually.

set -eu

REPO="vika2603/100x-cli"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="latest"

# ----- styling -------------------------------------------------------------
if [ -t 1 ] && [ -z "${NO_COLOR-}" ] && [ "${TERM-}" != "dumb" ]; then
	BOLD="$(printf '\033[1m')"
	DIM="$(printf '\033[2m')"
	RED="$(printf '\033[31m')"
	GREEN="$(printf '\033[32m')"
	YELLOW="$(printf '\033[33m')"
	CYAN="$(printf '\033[36m')"
	RESET="$(printf '\033[0m')"
else
	BOLD=""
	DIM=""
	RED=""
	GREEN=""
	YELLOW=""
	CYAN=""
	RESET=""
fi

step() { printf '%s==>%s %s\n' "$CYAN" "$RESET" "$1"; }
success() { printf '%s✓%s  %s\n' "$GREEN" "$RESET" "$1"; }
warn() { printf '%s!%s  %s\n' "$YELLOW" "$RESET" "$1"; }
fail() {
	printf '%s✗%s  %s\n' "$RED" "$RESET" "$1" >&2
	exit 1
}

# ----- args ----------------------------------------------------------------
usage() {
	cat <<EOF
${BOLD}Usage:${RESET} install.sh [--version vX.Y.Z] [--to <dir>]

  ${BOLD}--version${RESET}  Release tag to install (default: latest)
  ${BOLD}--to${RESET}       Install directory (default: \$HOME/.local/bin)
EOF
}

while [ $# -gt 0 ]; do
	case "$1" in
	--version)
		VERSION="$2"
		shift 2
		;;
	--to)
		INSTALL_DIR="$2"
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		fail "unknown arg: $1"
		;;
	esac
done

# ----- platform ------------------------------------------------------------
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$OS" in
linux | darwin) ;;
*)
	fail "unsupported OS: $OS  (use 'go install github.com/$REPO/cmd/100x@latest' or grab a binary from https://github.com/$REPO/releases)"
	;;
esac
case "$ARCH" in
x86_64 | amd64) ARCH="amd64" ;;
aarch64 | arm64) ARCH="arm64" ;;
*) fail "unsupported arch: $ARCH" ;;
esac

# ----- tooling -------------------------------------------------------------
if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -q "$1" -O "$2"; }
else
	fail "need curl or wget on PATH"
fi

if command -v sha256sum >/dev/null 2>&1; then
	sha() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
	sha() { shasum -a 256 "$1" | awk '{print $1}'; }
else
	fail "need sha256sum or shasum on PATH"
fi

# ----- banner --------------------------------------------------------------
printf '\n'
printf '  %s100x-cli installer%s\n' "$BOLD" "$RESET"
printf '  %shttps://github.com/%s%s\n' "$DIM" "$REPO" "$RESET"
printf '\n'

if [ "$VERSION" = "latest" ]; then
	BASE="https://github.com/$REPO/releases/latest/download"
else
	BASE="https://github.com/$REPO/releases/download/$VERSION"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# ----- fetch checksums -----------------------------------------------------
step "Resolving release"
fetch "$BASE/checksums.txt" "$TMP/checksums.txt"

ASSET="$(awk -v suffix="_${OS}_${ARCH}.tar.gz" 'index($2, suffix) {print $2; exit}' "$TMP/checksums.txt")"
EXPECTED="$(awk -v suffix="_${OS}_${ARCH}.tar.gz" 'index($2, suffix) {print $1; exit}' "$TMP/checksums.txt")"
if [ -z "$ASSET" ]; then
	fail "no asset matching ${OS}/${ARCH} in checksums.txt"
fi

# Derive the resolved tag from the asset name (matches goreleaser's pattern
# `<project>_<version>_<os>_<arch>.tar.gz`). Falls back to VERSION literal.
RESOLVED="$(printf '%s' "$ASSET" | sed -E 's/^[^_]+_(.+)_[^_]+_[^_]+\.tar\.gz$/\1/')"
[ -n "$RESOLVED" ] || RESOLVED="$VERSION"

# ----- download ------------------------------------------------------------
step "Downloading ${BOLD}${ASSET}${RESET}"
fetch "$BASE/$ASSET" "$TMP/$ASSET"

GOT="$(sha "$TMP/$ASSET")"
if [ "$GOT" != "$EXPECTED" ]; then
	fail "checksum mismatch for $ASSET
      expected: $EXPECTED
      got:      $GOT"
fi
success "Checksum verified  ${DIM}${EXPECTED}${RESET}"

# ----- extract & install ---------------------------------------------------
step "Extracting"
tar -xzf "$TMP/$ASSET" -C "$TMP" 100x

step "Installing to ${BOLD}${INSTALL_DIR}${RESET}"
mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMP/100x" "$INSTALL_DIR/100x"

# ----- summary -------------------------------------------------------------
printf '\n'
success "${BOLD}100x ${RESOLVED}${RESET} installed at ${BOLD}${INSTALL_DIR}/100x${RESET}"
case ":$PATH:" in
*":$INSTALL_DIR:"*) ;;
*)
	warn "${INSTALL_DIR} is not on PATH"
	printf '   Add this to your shell profile:\n'
	printf '     %sexport PATH="%s:$PATH"%s\n' "$DIM" "$INSTALL_DIR" "$RESET"
	;;
esac
printf '\n'
printf '  Try it:    %s100x --help%s\n' "$BOLD" "$RESET"
printf '  Add creds: %s100x profile add <name>%s\n' "$BOLD" "$RESET"
printf '\n'
