//go:build cgo && (linux || darwin)

package permissions

/*
#cgo CFLAGS: -I${SRCDIR}/../../native/permissions/include
#cgo LDFLAGS: ${SRCDIR}/../../native/permissions/lib/libretrocast_permissions.a
#cgo linux LDFLAGS: -ldl -lpthread -lm -lrt -lutil
#cgo darwin LDFLAGS: -liconv -lresolv
#include "permissions.h"
*/
import "C"

import (
	"runtime"

	"github.com/victorivanov/retrocast/internal/models"
)

const nativeABIVersion uint32 = C.RC_PERMISSIONS_ABI_VERSION

func nativeBase(version uint32, everyone int64, roles []models.Role) (Permission, error) {
	// Never pass model structs (strings/Go pointers) through cgo. These scalar
	// arrays contain no pointers and are borrowed synchronously by Rust.
	masks := make([]C.int64_t, len(roles))
	for i := range roles {
		masks[i] = C.int64_t(roles[i].Permissions)
	}
	var ptr *C.int64_t
	if len(masks) != 0 {
		ptr = &masks[0]
	}
	result := C.rc_permissions_base(C.uint32_t(version), C.int64_t(everyone), ptr, C.size_t(len(masks)))
	runtime.KeepAlive(masks)
	return nativeResult("base", int64(result.permissions), int32(result.status))
}

func nativeChannel(version uint32, base Permission, everyone *models.ChannelOverride, roles []models.ChannelOverride) (Permission, error) {
	overrides := make([]C.rc_permission_override, len(roles))
	for i := range roles {
		overrides[i].allow = C.int64_t(roles[i].Allow)
		overrides[i].deny = C.int64_t(roles[i].Deny)
	}
	var ptr *C.rc_permission_override
	if len(overrides) != 0 {
		ptr = &overrides[0]
	}
	var present C.uint32_t
	var allow, deny C.int64_t
	if everyone != nil {
		present = 1
		allow, deny = C.int64_t(everyone.Allow), C.int64_t(everyone.Deny)
	}
	result := C.rc_permissions_channel(C.uint32_t(version), C.int64_t(base), present, allow, deny, ptr, C.size_t(len(overrides)))
	runtime.KeepAlive(overrides)
	return nativeResult("channel", int64(result.permissions), int32(result.status))
}
