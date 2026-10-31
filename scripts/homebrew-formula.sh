#!/bin/sh
# Prints Formula/wariwari.rb for lostcoords/homebrew-tap from a published release's checksums.txt.
#
#   scripts/homebrew-formula.sh 0.1.0 > ../homebrew-tap/Formula/wariwari.rb
#
# WARIWARI_RELEASE_URL reads checksums.txt from another directory (file:// works), for testing.
set -eu

if [ $# -ne 1 ]; then
  echo "usage: $0 VERSION" >&2
  exit 2
fi
version=${1#v}
download="https://github.com/lostcoords/wariwari-cli/releases/download/v$version"
release=${WARIWARI_RELEASE_URL:-$download}

case "$release" in
  file://*) checksums=$(cat "${release#file://}/checksums.txt") ;;
  *) checksums=$(curl -fsSL "$release/checksums.txt") ;;
esac

# sha OS ARCH prints the archive's SHA-256, or fails when the release lacks it.
sha() {
  asset="wariwari_${version}_$1_$2.tar.gz"
  sum=$(printf '%s\n' "$checksums" | awk -v asset="$asset" '$2 == asset { print $1 }')
  if [ -z "$sum" ]; then
    echo "$asset is not in $release/checksums.txt" >&2
    exit 1
  fi
  echo "$sum"
}

# Resolve every checksum before printing, so a missing one prints no partial formula.
darwin_arm64=$(sha darwin arm64)
darwin_amd64=$(sha darwin amd64)
linux_arm64=$(sha linux arm64)
linux_amd64=$(sha linux amd64)

cat <<RUBY
class Wariwari < Formula
  desc "Split group expenses with WariWari from the command line and coding agents"
  homepage "https://github.com/lostcoords/wariwari-cli"
  license "MIT"

  on_macos do
    on_arm do
      url "$download/wariwari_${version}_darwin_arm64.tar.gz"
      sha256 "$darwin_arm64"
    end
    on_intel do
      url "$download/wariwari_${version}_darwin_amd64.tar.gz"
      sha256 "$darwin_amd64"
    end
  end

  on_linux do
    on_arm do
      url "$download/wariwari_${version}_linux_arm64.tar.gz"
      sha256 "$linux_arm64"
    end
    on_intel do
      url "$download/wariwari_${version}_linux_amd64.tar.gz"
      sha256 "$linux_amd64"
    end
  end

  def install
    bin.install "wariwari"
    doc.install "MIT-LICENSE", "README.md"
  end

  test do
    assert_match "wariwari #{version}", shell_output("#{bin}/wariwari --version")
  end
end
RUBY
