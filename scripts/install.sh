#!/usr/bin/env sh
set -eu

REPO="${OPENAGENTSGATE_REPO:-arnesssr/OpenAgentsGate}"
BIN_NAME="openagentsgate"
BIN_DIR="${OPENAGENTSGATE_BIN_DIR:-$HOME/.local/bin}"
VERSION="${OPENAGENTSGATE_VERSION:-latest}"
SOURCE="${OPENAGENTSGATE_INSTALL_SOURCE:-auto}"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
}

download() {
  url="$1"
  out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$out"
  else
    echo "missing required command: curl or wget" >&2
    exit 1
  fi
}

sha256_file() {
  file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    echo ""
  fi
}

resolve_version() {
  if [ "$VERSION" != "latest" ]; then
    echo "$VERSION"
    return
  fi
  if download "https://api.github.com/repos/$REPO/releases/latest" "$TMP_DIR/latest.json" 2>/dev/null; then
    sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' "$TMP_DIR/latest.json" | head -n 1
  fi
}

install_from_go() {
  need go
  mkdir -p "$BIN_DIR"
  module="github.com/$REPO/cmd/openagentsgate@$VERSION"
  if [ "$VERSION" = "latest" ]; then
    module="github.com/$REPO/cmd/openagentsgate@latest"
  fi
  echo "installing with go install: $module"
  date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  ldflags="-s -w"
  ldflags="$ldflags -X github.com/arnesssr/OpenAgentsGate/internal/buildinfo.Version=$VERSION"
  ldflags="$ldflags -X github.com/arnesssr/OpenAgentsGate/internal/buildinfo.Commit=go-install"
  ldflags="$ldflags -X github.com/arnesssr/OpenAgentsGate/internal/buildinfo.Date=$date"
  GOBIN="$BIN_DIR" go install -ldflags "$ldflags" "$module"
  echo "installed $BIN_NAME to $BIN_DIR/$BIN_NAME"
}

main() {
  if [ "$SOURCE" = "go" ]; then
    install_from_go
    return
  fi

  os="$(detect_os)"
  arch="$(detect_arch)"
  tag="$(resolve_version)"
  if [ -z "$tag" ]; then
    if [ "$SOURCE" = "auto" ] && [ "$VERSION" = "latest" ]; then
      echo "no release found; falling back to go install"
      install_from_go
      return
    fi
    echo "could not resolve release version" >&2
    exit 1
  fi

  need tar
  ext="tar.gz"
  archive="${BIN_NAME}_${tag#v}_${os}_${arch}.${ext}"
  base="https://github.com/$REPO/releases/download/$tag"
  if ! download "$base/$archive" "$TMP_DIR/$archive"; then
    if [ "$SOURCE" = "auto" ]; then
      echo "release asset unavailable; falling back to go install"
      VERSION="$tag"
      install_from_go
      return
    fi
    echo "could not download release asset: $archive" >&2
    exit 1
  fi
  if download "$base/checksums.txt" "$TMP_DIR/checksums.txt" 2>/dev/null; then
    expected="$(awk -v f="$archive" '$2 == f {print $1}' "$TMP_DIR/checksums.txt")"
    actual="$(sha256_file "$TMP_DIR/$archive")"
    if [ -n "$expected" ] && [ -n "$actual" ] && [ "$expected" != "$actual" ]; then
      echo "checksum mismatch for $archive" >&2
      exit 1
    fi
  fi

  mkdir -p "$BIN_DIR"
  tar -xzf "$TMP_DIR/$archive" -C "$TMP_DIR"
  install -m 0755 "$TMP_DIR/$BIN_NAME" "$BIN_DIR/$BIN_NAME"
  echo "installed $BIN_NAME $tag to $BIN_DIR/$BIN_NAME"
}

main "$@"
