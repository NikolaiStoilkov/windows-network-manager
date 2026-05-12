#!/usr/bin/env bash
# build_console.sh
# Cross-compiles a Windows console binary (shows a terminal window on launch).
# Output: network_manager.exe

set -euo pipefail

OUTPUT="network_manager.exe"

echo "[*] Building console binary → ${OUTPUT}"

GOOS=windows GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w" \
  -o "${OUTPUT}" \
  .

echo "[+] Done: ${OUTPUT}"

