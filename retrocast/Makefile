.PHONY: build test run clean native native-test permission-test package-check cache-check
.DEFAULT_GOAL := build

CGO_ENABLED ?= 1
export CGO_ENABLED

native:
	./scripts/build-native.sh

native-test:
	cargo test --manifest-path native/permissions/Cargo.toml --locked --offline
	cargo clippy --manifest-path native/permissions/Cargo.toml --locked --offline --all-targets -- -D warnings
	cargo fmt --manifest-path native/permissions/Cargo.toml -- --check

permission-test: native
	go test -json -count=1 ./scripts/nativefingerprint
	go test -json -race -count=1 ./internal/permissions ./internal/service

package-check: native
	./scripts/test-native-package.sh

# Focused cgo cache regression with synthetic C archives; no services or Rust
# compilation. The full package-check still includes this regression as well.
cache-check:
	./scripts/test-native-package.sh --warm-cache-only

build: native
	go build -ldflags='-linkmode=external' -o bin/retrocast ./cmd/retrocast

test: native
	go test -v -race -count=1 ./...

run: build
	./bin/retrocast

clean:
	rm -rf bin/
