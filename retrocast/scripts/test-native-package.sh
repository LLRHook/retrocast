#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
repo=$(pwd)
mkdir -p bin
stage=$(mktemp -d "$repo/bin/native-package.XXXXXX")
mkdir -p "$stage/runtime"

# These are test-only executables; the application/server is never started.
go build -ldflags='-linkmode=external' -o "$stage/permission-probe" ./scripts/testdata/permissionprobe
go tool nm "$stage/permission-probe" > "$stage/symbols.log"
grep -q 'rc_permissions_base' "$stage/symbols.log"
grep -q 'rc_permissions_channel' "$stage/symbols.log"
(cd "$stage/runtime" && "$stage/permission-probe")

# Assemble only the source leaves needed by the real PermissionChecker in a
# synthetic module. This is not a checkout, does not access services, and has no
# external dependencies. Never replace the real application's native archive.
prepare_module() {
    module="$stage/$1"
    mkdir -p "$module/internal/models" "$module/internal/permissions" \
        "$module/internal/service" "$module/internal/database" \
        "$module/native/permissions/include" "$module/cmd/probe"
    printf 'module github.com/victorivanov/retrocast\n\ngo 1.25.7\n' > "$module/go.mod"
    cp internal/models/*.go "$module/internal/models/"
    for source in internal/permissions/*.go; do
        case "$source" in *_test.go) ;; *) cp "$source" "$module/internal/permissions/" ;; esac
    done
    cp internal/service/permission.go internal/service/errors.go "$module/internal/service/"
    cp internal/database/repositories.go "$module/internal/database/"
    cp native/permissions/include/permissions.h "$module/native/permissions/include/"
    cp scripts/testdata/permissionprobe/main.go "$module/cmd/probe/"
}

stamp_module() {
    go run ./scripts/nativefingerprint "$module/native/permissions/lib/libretrocast_permissions.a" "$module/internal/permissions/native_fingerprint.go"
}

prepare_module missing
if (cd "$module" && GOWORK=off CGO_ENABLED=1 go build -o probe ./cmd/probe) > "$stage/missing.log" 2>&1; then
    echo "FAIL: a missing required Rust archive built successfully" >&2
    exit 1
fi
if ! grep -q 'libretrocast_permissions.a' "$stage/missing.log" || \
    ! grep -Eq 'No such file|no such file|cannot find|cannot open' "$stage/missing.log"; then
    cat "$stage/missing.log" >&2
    exit 1
fi
echo "PASS missing archive: build fails at required static library"

if (cd "$module" && GOWORK=off CGO_ENABLED=0 go build -o probe ./cmd/probe) > "$stage/cgo-disabled.log" 2>&1; then
    echo "FAIL: CGO_ENABLED=0 silently selected a fallback" >&2
    exit 1
fi
if ! grep -q 'Rust_permissions_require_CGO_ENABLED_1' "$stage/cgo-disabled.log"; then
    cat "$stage/cgo-disabled.log" >&2
    exit 1
fi
echo "PASS CGO_ENABLED=0: unsupported build fails explicitly"

for status in 1 2 3 99; do
    for scope in base channel-only; do
        prepare_module "fault-$status-$scope"
        mkdir -p "$module/native/permissions/lib"
        if [ "$scope" = channel-only ]; then base_ok=1; else base_ok=0; fi
        "${CC:-cc}" -I native/permissions/include -DRC_TEST_STATUS="$status" -DRC_TEST_BASE_OK="$base_ok" \
            -c scripts/testdata/nativefault/permissions.c -o "$module/fault.o"
        ar crs "$module/native/permissions/lib/libretrocast_permissions.a" "$module/fault.o"
        stamp_module
        (cd "$module" && GOWORK=off CGO_ENABLED=1 go build -ldflags='-linkmode=external' -o "$stage/fault-probe-$status-$scope" ./cmd/probe)
        (cd "$stage/runtime" && "$stage/fault-probe-$status-$scope" "$status" "$scope")
    done
done

# Replace the archive at the SAME path, leaving the host code, output path and
# shared Go caches untouched. Without native-content fingerprinting the second
# build can reuse the first native implementation and this regression fails.
prepare_module warm-cache
mkdir -p "$module/native/permissions/lib"
for status in 1 2; do
    "${CC:-cc}" -I native/permissions/include -DRC_TEST_STATUS="$status" -DRC_TEST_BASE_OK=0 \
        -c scripts/testdata/nativefault/permissions.c -o "$module/fault.o"
    ar crs "$module/native/permissions/lib/libretrocast_permissions.a" "$module/fault.o"
    stamp_module
    (cd "$module" && GOWORK=off CGO_ENABLED=1 go build -ldflags='-linkmode=external' -o "$stage/warm-cache-probe" ./cmd/probe)
    (cd "$stage/runtime" && "$stage/warm-cache-probe" "$status" base)
done
echo "PASS stable-path warm cache: replacing the native archive changes executed authorization code"
echo "Native package evidence retained in $stage"
