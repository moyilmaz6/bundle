#!/bin/sh
set -eu

prefix=/usr
rm -f "$prefix/lib/bundle-runtime/bundle-runtime-runner"
rmdir "$prefix/lib/bundle-runtime" 2>/dev/null || true
rm -f "$prefix/share/applications/bundle-runtime.desktop"
rm -f "$prefix/share/mime/packages/bundl-mime.xml"
update-mime-database "$prefix/share/mime"
update-desktop-database "$prefix/share/applications" 2>/dev/null || true
