#!/usr/bin/env bash
# build_gui.sh
# Cross-compiles a Windows GUI binary (no console / terminal window on launch).
# Output: network_manager_no_console.exe

set -euo pipefail

OUTPUT="network_manager_no_console.exe"

echo "[*] Building (no console) binary → ${OUTPUT}"

GOOS=windows GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w -H windowsgui" \
  -o "${OUTPUT}" \
  .

echo "[+] Done: ${OUTPUT}"

