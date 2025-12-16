#!/usr/bin/env sh
set -eu

REPO="datagendev/datagen-cli"
BINARY="datagen"

say() {
  printf "%s\n" "$*"
}

die() {
  printf "error: %s\n" "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

get_os() {
  os="$(uname -s 2>/dev/null || echo unknown)"
  case "$os" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *) die "unsupported OS: $os (supported: macOS, Linux)" ;;
  esac
}

get_arch() {
  arch="$(uname -m 2>/dev/null || echo unknown)"
  case "$arch" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) die "unsupported architecture: $arch (supported: amd64, arm64)" ;;
  esac
}

pick_install_dir() {
  if [ "${DATAGEN_INSTALL_DIR:-}" != "" ]; then
    echo "$DATAGEN_INSTALL_DIR"
    return 0
  fi

  if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
    echo "/usr/local/bin"
    return 0
  fi

  echo "${HOME}/.local/bin"
}

is_in_path() {
  case ":${PATH:-}:" in
    *":$1:"*) return 0 ;;
    *) return 1 ;;
  esac
}

make_tmpdir() {
  if tmp="$(mktemp -d 2>/dev/null)"; then
    echo "$tmp"
    return 0
  fi
  mktemp -d -t datagen
}

download() {
  url="$1"
  out="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out"
    return 0
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO "$out" "$url"
    return 0
  fi

  die "need curl or wget to download"
}

main() {
  need_cmd uname
  need_cmd chmod
  need_cmd mkdir

  os="$(get_os)"
  arch="$(get_arch)"
  asset="${BINARY}-${os}-${arch}"

  version="${DATAGEN_VERSION:-latest}"
  if [ "$version" = "latest" ]; then
    url="https://github.com/${REPO}/releases/latest/download/${asset}"
  else
    url="https://github.com/${REPO}/releases/download/${version}/${asset}"
  fi

  install_dir="$(pick_install_dir)"
  mkdir -p "$install_dir"

  tmpdir="$(make_tmpdir)"
  trap 'rm -rf "$tmpdir"' EXIT INT TERM

  say "Downloading ${asset} (${version})..."
  download "$url" "${tmpdir}/${BINARY}"
  chmod +x "${tmpdir}/${BINARY}"

  if command -v install >/dev/null 2>&1; then
    install -m 0755 "${tmpdir}/${BINARY}" "${install_dir}/${BINARY}"
  else
    mv "${tmpdir}/${BINARY}" "${install_dir}/${BINARY}"
  fi

  say "Installed: ${install_dir}/${BINARY}"
  if ! is_in_path "$install_dir"; then
    say ""
    say "Add it to your PATH (example):"
    say "  echo 'export PATH=\"${install_dir}:\\$PATH\"' >> ~/.zshrc"
    say "  source ~/.zshrc"
  fi

  say ""
  say "Verify:"
  say "  ${BINARY} --help"
}

main "$@"

