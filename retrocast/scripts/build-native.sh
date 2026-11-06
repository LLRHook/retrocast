#!/bin/sh
set -eu
cd "$(dirname "$0")/.."

case "$(uname -s)" in
    Linux|Darwin) ;;
    *) echo "Rust permissions currently support native Linux and macOS builds only" >&2; exit 1 ;;
esac
if [ "$(go env CGO_ENABLED)" != 1 ]; then
    echo "Rust permissions require CGO_ENABLED=1 and a host C compiler" >&2
    exit 1
fi
if [ "$(go env GOOS)" != "$(go env GOHOSTOS)" ] || [ "$(go env GOARCH)" != "$(go env GOHOSTARCH)" ] || [ -n "${CARGO_BUILD_TARGET:-}" ]; then
    echo "Build Rust and Go on the target host; cross-compilation is not configured" >&2
    exit 1
fi

# Respect a shared caller-owned cache. The stable lib path is required by cgo;
# no compiled library is committed or searched for dynamically at runtime.
: "${CARGO_TARGET_DIR:=$(pwd)/native/permissions/target}"
export CARGO_TARGET_DIR
cargo build --manifest-path native/permissions/Cargo.toml --locked --offline --release
mkdir -p native/permissions/lib
cp "$CARGO_TARGET_DIR/release/libretrocast_permissions.a" native/permissions/lib/libretrocast_permissions.a
# Go's build cache does not track linked C archives. Make their content a Go
# source input so Rust-only changes invalidate the cgo package and its callers.
go run ./scripts/nativefingerprint native/permissions/lib/libretrocast_permissions.a internal/permissions/native_fingerprint.go
