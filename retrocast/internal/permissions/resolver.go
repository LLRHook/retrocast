package permissions

import (
	"fmt"

	"github.com/victorivanov/retrocast/internal/models"
)

// NativeError reports an ABI/argument/panic status from the required Rust engine.
// Permission results accompanying a nonzero status are never used.
type NativeError struct {
	Operation string
	Status    int32
}

func (e *NativeError) Error() string {
	return fmt.Sprintf("Rust permission resolver %s failed (ABI status %d)", e.Operation, e.Status)
}

// ComputeBasePermissions computes the union of everyone and member roles in Rust,
// returning PermAll when Administrator is set. It panics on a native fault because
// its historical signature cannot return an error; use the checked form in services.
func ComputeBasePermissions(everyoneRole models.Role, memberRoles []models.Role) Permission {
	return mustPermission(ComputeBasePermissionsChecked(everyoneRole, memberRoles))
}

// ComputeBasePermissionsChecked is the error-returning form used by request paths.
func ComputeBasePermissionsChecked(everyoneRole models.Role, memberRoles []models.Role) (Permission, error) {
	return nativeBase(nativeABIVersion, everyoneRole.Permissions, memberRoles)
}

// ComputeChannelPermissions resolves the complete channel mask in Rust: base
// Administrator bypass, everyone deny/allow, then aggregated role deny/allow.
// It panics on a native fault; use the checked form in services.
func ComputeChannelPermissions(basePerms Permission, everyoneOverride *models.ChannelOverride, roleOverrides []models.ChannelOverride) Permission {
	return mustPermission(ComputeChannelPermissionsChecked(basePerms, everyoneOverride, roleOverrides))
}

// ComputeChannelPermissionsChecked never treats a transport fault as a mask,
// including when the requested mask is zero or the base has Administrator.
func ComputeChannelPermissionsChecked(basePerms Permission, everyoneOverride *models.ChannelOverride, roleOverrides []models.ChannelOverride) (Permission, error) {
	return nativeChannel(nativeABIVersion, basePerms, everyoneOverride, roleOverrides)
}

func nativeResult(operation string, mask int64, status int32) (Permission, error) {
	if status != 0 {
		return 0, &NativeError{Operation: operation, Status: status}
	}
	return Permission(mask), nil
}

func mustPermission(mask Permission, err error) Permission {
	if err != nil {
		panic(err)
	}
	return mask
}
