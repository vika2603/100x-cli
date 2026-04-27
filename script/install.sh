#!/usr/bin/env sh
# Install 100x-cli by downloading a release archive that matches the host
# OS/arch, verifying its SHA-256, and copying the binary into a directory
# on PATH. POSIX shell, no third-party deps. Linux and macOS only; on
# Windows, use `go install` or download an asset manually.

set -eu

REPO="vika2603/100x-cli"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="latest"

usage() {
	cat <<EOF
Usage: install.sh [--version vX.Y.Z] [--to <dir>]

  --version  Release tag to install (default: latest)
  --to       Install directory (default: \$HOME/.local/bin)
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
		echo "unknown arg: $1" >&2
		usage >&2
		exit 2
		;;
	esac
done

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$OS" in
linux | darwin) ;;
*)
	echo "unsupported OS: $OS (use 'go install github.com/$REPO/cmd/100x@latest' or download from https://github.com/$REPO/releases)" >&2
	exit 1
	;;
esac
case "$ARCH" in
x86_64 | amd64) ARCH="amd64" ;;
aarch64 | arm64) ARCH="arm64" ;;
*)
	echo "unsupported arch: $ARCH" >&2
	exit 1
	;;
esac

if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -q "$1" -O "$2"; }
else
	echo "need curl or wget on PATH" >&2
	exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
	sha() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
	sha() { shasum -a 256 "$1" | awk '{print $1}'; }
else
	echo "need sha256sum or shasum on PATH" >&2
	exit 1
fi

if [ "$VERSION" = "latest" ]; then
	BASE="https://github.com/$REPO/releases/latest/download"
else
	BASE="https://github.com/$REPO/releases/download/$VERSION"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "fetching checksums.txt"
fetch "$BASE/checksums.txt" "$TMP/checksums.txt"

ASSET="$(awk -v suffix="_${OS}_${ARCH}.tar.gz" 'index($2, suffix) {print $2; exit}' "$TMP/checksums.txt")"
EXPECTED="$(awk -v suffix="_${OS}_${ARCH}.tar.gz" 'index($2, suffix) {print $1; exit}' "$TMP/checksums.txt")"
if [ -z "$ASSET" ]; then
	echo "no asset matching ${OS}/${ARCH} in checksums.txt" >&2
	exit 1
fi

echo "downloading $ASSET"
fetch "$BASE/$ASSET" "$TMP/$ASSET"

GOT="$(sha "$TMP/$ASSET")"
if [ "$GOT" != "$EXPECTED" ]; then
	echo "checksum mismatch for $ASSET" >&2
	echo "  expected: $EXPECTED" >&2
	echo "  got:      $GOT" >&2
	exit 1
fi

echo "extracting"
tar -xzf "$TMP/$ASSET" -C "$TMP" 100x

mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMP/100x" "$INSTALL_DIR/100x"

echo "installed $INSTALL_DIR/100x"
case ":$PATH:" in
*":$INSTALL_DIR:"*) ;;
*) echo "note: $INSTALL_DIR is not on PATH" ;;
esac
