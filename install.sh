#!/usr/bin/env sh
set -eu

BIN_DIR="${GOBIN:-${GOPATH:-$HOME/go}/bin}"
mkdir -p "$BIN_DIR"
go build -o "$BIN_DIR/devpkgs" .
echo "Installed devpkgs to $BIN_DIR/devpkgs"
