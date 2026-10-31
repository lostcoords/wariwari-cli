#!/bin/sh
# Installs the wariwari CLI into ~/.local/bin, checked against the release's checksums.txt.
# Served at https://wariwari.jp/install.sh from github.com/lostcoords/wariwari-cli.
#
#   curl -fsSL https://wariwari.jp/install.sh | sh
#   curl -fsSL https://wariwari.jp/install.sh | WARIWARI_VERSION=v0.1.0 sh
#
# WARIWARI_BIN_DIR changes the install directory. WARIWARI_RELEASE_URL points at a directory
# holding checksums.txt and the archives, for testing a local build.
set -eu

REPO="lostcoords/wariwari-cli"
BIN="${WARIWARI_BIN_DIR:-$HOME/.local/bin}"

# digest FILE prints the file's SHA-256.
digest() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  else
    shasum -a 256 "$1" | cut -d' ' -f1
  fi
}

download() {
  case "$1" in
    file://*) cp "${1#file://}" "$2" ;;
    *) curl -fsSL "$1" -o "$2" ;;
  esac
}

# path_hint prints how to put $BIN on PATH when it is not there yet.
path_hint() {
  case ":$PATH:" in
    *":$BIN:"*) return 0 ;;
  esac
  shown=$BIN
  case "$BIN" in
    "$HOME"/*) shown="\$HOME${BIN#"$HOME"}" ;;
  esac
  echo
  echo "Add $shown to PATH:"
  case "${SHELL##*/}" in
    fish) echo "  fish_add_path $shown" ;;
    zsh) echo "  echo 'export PATH=\"$shown:\$PATH\"' >> ~/.zshrc" ;;
    bash) echo "  echo 'export PATH=\"$shown:\$PATH\"' >> ~/.bashrc" ;;
    *) echo "  export PATH=\"$shown:\$PATH\"" ;;
  esac
}

main() {
  case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) echo "wariwari: no build for $(uname -s); use go install instead" >&2; exit 1 ;;
  esac
  case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) echo "wariwari: no build for $(uname -m); use go install instead" >&2; exit 1 ;;
  esac

  if [ -n "${WARIWARI_RELEASE_URL:-}" ]; then
    release=$WARIWARI_RELEASE_URL
  elif [ -n "${WARIWARI_VERSION:-}" ]; then
    release="https://github.com/$REPO/releases/download/v${WARIWARI_VERSION#v}"
  else
    release="https://github.com/$REPO/releases/latest/download"
  fi

  work=$(mktemp -d)
  trap 'rm -rf "$work"' EXIT

  # GoReleaser names archives wariwari_<version without v>_<os>_<arch>.tar.gz; the checksum
  # file lists them, so it also tells us the version without asking the GitHub API.
  download "$release/checksums.txt" "$work/checksums.txt"
  line=$(grep "  wariwari_.*_${os}_${arch}\.tar\.gz\$" "$work/checksums.txt" || true)
  if [ -z "$line" ]; then
    echo "wariwari: no $os/$arch archive in $release/checksums.txt" >&2
    exit 1
  fi
  expected=${line%% *}
  asset=${line##* }

  download "$release/$asset" "$work/$asset"
  if [ "$(digest "$work/$asset")" != "$expected" ]; then
    echo "wariwari: checksum mismatch for $asset; nothing installed" >&2
    exit 1
  fi

  tar -xzf "$work/$asset" -C "$work" wariwari
  mkdir -p "$BIN"
  install -m 755 "$work/wariwari" "$BIN/wariwari"
  echo "installed $("$BIN/wariwari" --version) to $BIN/wariwari"

  path_hint
  echo
  echo "Next: wariwari auth login wari_... (create a key under Account -> API keys),"
  echo "then wariwari skill install to give your coding agent the WariWari skill."
  echo "Upgrade later with: wariwari upgrade"
}

main "$@"
