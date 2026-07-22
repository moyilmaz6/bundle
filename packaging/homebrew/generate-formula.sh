#!/usr/bin/env bash
set -euo pipefail

ver="${1:?usage: generate-formula.sh <version> <checksums.txt>}"
sums="${2:?usage: generate-formula.sh <version> <checksums.txt>}"

sha() {
  local file="bundle_${ver}_$1.tar.gz" hash
  hash="$(awk -v f="$file" '$2 == f {print $1}' "$sums")"
  [ -n "$hash" ] || { echo "no checksum for $file in $sums" >&2; exit 1; }
  printf '%s' "$hash"
}

base="https://github.com/moyilmaz6/bundle/releases/download/v${ver}"

cat <<RUBY
class Bundle < Formula
  desc "Package a Go web server and its web UI into a native desktop app"
  homepage "https://github.com/moyilmaz6/bundle"
  version "${ver}"

  on_macos do
    on_arm do
      url "${base}/bundle_${ver}_darwin-arm64.tar.gz"
      sha256 "$(sha darwin-arm64)"
    end
    on_intel do
      url "${base}/bundle_${ver}_darwin-amd64.tar.gz"
      sha256 "$(sha darwin-amd64)"
    end
  end

  on_linux do
    on_arm do
      url "${base}/bundle_${ver}_linux-arm64.tar.gz"
      sha256 "$(sha linux-arm64)"
    end
    on_intel do
      url "${base}/bundle_${ver}_linux-amd64.tar.gz"
      sha256 "$(sha linux-amd64)"
    end
  end

  def install
    bin.install "bundle"
    generate_completions_from_executable(bin/"bundle", "completion")
  end

  test do
    system bin/"bundle", "--help"
  end
end
RUBY
