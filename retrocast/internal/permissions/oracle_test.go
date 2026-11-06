// Frozen independent Go oracle from resolver.go at 6bc385e2634d1ac2f4e5a1c220b860a1a92a05f8.
// Original SHA-256: a0dd704f6ce10b92c3bba684616b9a6fd807d851bd113b8656efbc550b630638
// Only the two function names were changed. Never imported by production.
package permissions

import "github.com/victorivanov/retrocast/internal/models"

// frozenBasePermissions computes guild-level permissions for a member.
//  1. Start with the @everyone role permissions.
//  2. OR all the member's assigned role permissions.
//  3. If the result includes ADMINISTRATOR, return PermAll.
func frozenBasePermissions(everyoneRole models.Role, memberRoles []models.Role) Permission {
	perms := Permission(everyoneRole.Permissions)

	for _, role := range memberRoles {
		perms = perms.Add(Permission(role.Permissions))
	}

	if perms.Has(PermAdministrator) {
		return PermAll
	}
	return perms
}

// frozenChannelPermissions applies channel-specific overrides to base permissions.
//  1. Start with base permissions.
//  2. If ADMINISTRATOR, return PermAll (skip overrides).
//  3. Apply @everyone channel override: deny first, then allow.
//  4. Apply role overrides: OR all role allows, OR all role denies, then deny, then allow.
//  5. Return final permissions.
func frozenChannelPermissions(basePerms Permission, everyoneOverride *models.ChannelOverride, roleOverrides []models.ChannelOverride) Permission {
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
