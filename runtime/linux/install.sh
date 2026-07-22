#!/bin/sh
set -eu

prefix=/usr
install -Dm755 bundle-runtime-runner "$prefix/lib/bundle-runtime/bundle-runtime-runner"
install -Dm644 bundle-runtime.desktop "$prefix/share/applications/bundle-runtime.desktop"
install -Dm644 bundl-mime.xml "$prefix/share/mime/packages/bundl-mime.xml"
update-mime-database "$prefix/share/mime"
update-desktop-database "$prefix/share/applications" 2>/dev/null || true
