#!/usr/bin/env bash
set -euo pipefail

ver="20260616"
url="https://github.com/mstorsjo/llvm-mingw/releases/download/${ver}/llvm-mingw-${ver}-ucrt-aarch64.zip"

curl -fL -o "$RUNNER_TEMP/llvm-mingw.zip" "$url"
7z x "$RUNNER_TEMP/llvm-mingw.zip" -o"$RUNNER_TEMP" >/dev/null

bindir="$RUNNER_TEMP/llvm-mingw-${ver}-ucrt-aarch64/bin"
echo "$bindir" >> "$GITHUB_PATH"
echo "CC=aarch64-w64-mingw32-clang" >> "$GITHUB_ENV"
echo "CXX=aarch64-w64-mingw32-clang++" >> "$GITHUB_ENV"
echo "installed llvm-mingw ${ver}; CC=aarch64-w64-mingw32-clang"
