#!/usr/bin/env sh
set -eu

BIN_DIR="${GOBIN:-${GOPATH:-$HOME/go}/bin}"
mkdir -p "$BIN_DIR"
go build -o "$BIN_DIR/pkgview" .
echo "Installed pkgview to $BIN_DIR/pkgview"
