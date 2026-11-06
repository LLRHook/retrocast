//go:build !cgo || (!linux && !darwin)

package permissions

// Deliberate build failure: there is no pure-Go or unsupported-host fallback.
// See README: build the Rust archive with make native, then use CGO_ENABLED=1.
var _ = Rust_permissions_require_CGO_ENABLED_1_on_Linux_or_macOS
