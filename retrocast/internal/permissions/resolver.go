package permissions

import "github.com/victorivanov/retrocast/internal/models"

// ComputeBasePermissions computes guild-level permissions for a member.
//  1. Start with the @everyone role permissions.
//  2. OR all the member's assigned role permissions.
//  3. If the result includes ADMINISTRATOR, return PermAll.
func ComputeBasePermissions(everyoneRole models.Role, memberRoles []models.Role) Permission {
	perms := Permission(everyoneRole.Permissions)

	for _, role := range memberRoles {
		perms = perms.Add(Permission(role.Permissions))
	}

	if perms.Has(PermAdministrator) {
		return PermAll
	}
	return perms
}

// ComputeChannelPermissions applies channel-specific overrides to base permissions.
//  1. Start with base permissions.
//  2. If ADMINISTRATOR, return PermAll (skip overrides).
//  3. Apply @everyone channel override: deny first, then allow.
//  4. Apply role overrides: OR all role allows, OR all role denies, then deny, then allow.
//  5. Return final permissions.
func ComputeChannelPermissions(basePerms Permission, everyoneOverride *models.ChannelOverride, roleOverrides []models.ChannelOverride) Permission {
	if basePerms.Has(PermAdministrator) {
		return PermAll
	}

	perms := basePerms

	// Apply @everyone channel override first.
	if everyoneOverride != nil {
		perms = perms.Remove(Permission(everyoneOverride.Deny))
		perms = perms.Add(Permission(everyoneOverride.Allow))
	}

	// Aggregate role overrides: OR all allows together, OR all denies together.
	var roleAllow, roleDeny Permission
	for _, o := range roleOverrides {
		roleAllow = roleAllow.Add(Permission(o.Allow))
		roleDeny = roleDeny.Add(Permission(o.Deny))
	}

	perms = perms.Remove(roleDeny)
	perms = perms.Add(roleAllow)

	return perms
}
