#!/usr/bin/env bash
# Build helper for magnet2video. Replaces the old Makefile.
#
# Usage:
#   ./build.sh [command]
#
# Run `./build.sh help` for the full list of commands.

set -euo pipefail

BINARY="magnet2video"
BIN_DIR="bin"
GO="${GO:-go}"
GOFLAGS="${GOFLAGS:-}"
LDFLAGS="${LDFLAGS:--s -w}"

ensure_bin_dir() {
    mkdir -p "$BIN_DIR"
}

build_one() {
    local suffix="$1"
    local out="$BIN_DIR/$BINARY"
    [[ -n "$suffix" ]] && out="$out-$suffix"
    # shellcheck disable=SC2086
    "$GO" build $GOFLAGS -ldflags "$LDFLAGS" -o "$out" .
    echo "  built $out"
}

cmd_split() {
    ensure_bin_dir
    echo "Building server + worker binaries..."
    build_one server
    build_one worker
    echo
    echo "Done. Run with:"
    echo "  ./$BIN_DIR/$BINARY-server -mode=server"
    echo "  ./$BIN_DIR/$BINARY-worker -mode=worker"
    echo
    echo "Tip: both binaries are identical; -mode flag decides behaviour."
}

cmd_mono() {
    ensure_bin_dir
    echo "Building single all-in-one binary..."
    build_one ""
    echo "Done. Defaults to -mode=all."
}

cmd_server() {
    ensure_bin_dir
    build_one server
}

cmd_worker() {
    ensure_bin_dir
    build_one worker
}

cmd_test() { "$GO" test ./...; }
cmd_vet()  { "$GO" vet ./...; }
cmd_fmt()  { gofmt -w .; }

cmd_run_all()    { "$GO" run . -mode=all; }
cmd_run_server() { "$GO" run . -mode=server; }
cmd_run_worker() { "$GO" run . -mode=worker; }

cmd_clean() {
    rm -rf "$BIN_DIR"
    echo "Removed $BIN_DIR/"
}

cmd_help() {
    cat <<EOF
magnet2video build script

Usage: ./build.sh [command]

Build commands:
  all | split     Build server + worker binaries (default)
  mono            Build single all-in-one binary
  server          Build only the server binary
  worker          Build only the worker binary
  clean           Remove the $BIN_DIR/ directory

Run commands (via \`go run\`):
  run-all         Run in mode=all
  run-server      Run in mode=server
  run-worker      Run in mode=worker

Quality commands:
  test            go test ./...
  vet             go vet ./...
  fmt             gofmt -w .

Other:
  help            Show this message

Environment overrides:
  GO=/path/to/go  Go binary to use (default: go)
  GOFLAGS="..."   Extra flags passed to \`go build\`
  LDFLAGS="..."   Override linker flags (default: -s -w)
EOF
}

main() {
    local cmd="${1:-all}"
    case "$cmd" in
        all|split)   cmd_split ;;
        mono)        cmd_mono ;;
        server)      cmd_server ;;
        worker)      cmd_worker ;;
        test)        cmd_test ;;
        vet)         cmd_vet ;;
        fmt)         cmd_fmt ;;
        run-all)     cmd_run_all ;;
        run-server)  cmd_run_server ;;
        run-worker)  cmd_run_worker ;;
        clean)       cmd_clean ;;
        help|-h|--help) cmd_help ;;
        *)
            echo "Unknown command: $cmd" >&2
            echo >&2
            cmd_help >&2
            exit 2
            ;;
    esac
}

main "$@"
