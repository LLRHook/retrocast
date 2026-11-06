// Frozen resolver from 6bc385e2634d1ac2f4e5a1c220b860a1a92a05f8 for independent service-policy comparison.
// Only package/imports, type qualification, and function names changed.
package service

import (
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// frozenServiceBasePermissions computes guild-level permissions for a member.
//  1. Start with the @everyone role permissions.
//  2. OR all the member's assigned role permissions.
//  3. If the result includes ADMINISTRATOR, return permissions.PermAll.
func frozenServiceBasePermissions(everyoneRole models.Role, memberRoles []models.Role) permissions.Permission {
	perms := permissions.Permission(everyoneRole.Permissions)

	for _, role := range memberRoles {
		perms = perms.Add(permissions.Permission(role.Permissions))
	}

	if perms.Has(permissions.PermAdministrator) {
		return permissions.PermAll
	}
	return perms
}

// frozenServiceChannelPermissions applies channel-specific overrides to base permissions.
//  1. Start with base permissions.
//  2. If ADMINISTRATOR, return permissions.PermAll (skip overrides).
//  3. Apply @everyone channel override: deny first, then allow.
//  4. Apply role overrides: OR all role allows, OR all role denies, then deny, then allow.
//  5. Return final permissions.
func frozenServiceChannelPermissions(basePerms permissions.Permission, everyoneOverride *models.ChannelOverride, roleOverrides []models.ChannelOverride) permissions.Permission {
	if basePerms.Has(permissions.PermAdministrator) {
		return permissions.PermAll
	}

	perms := basePerms

	// Apply @everyone channel override first.
	if everyoneOverride != nil {
		perms = perms.Remove(permissions.Permission(everyoneOverride.Deny))
		perms = perms.Add(permissions.Permission(everyoneOverride.Allow))
	}

	// Aggregate role overrides: OR all allows together, OR all denies together.
	var roleAllow, roleDeny permissions.Permission
	for _, o := range roleOverrides {
		roleAllow = roleAllow.Add(permissions.Permission(o.Allow))
		roleDeny = roleDeny.Add(permissions.Permission(o.Deny))
	}

	perms = perms.Remove(roleDeny)
	perms = perms.Add(roleAllow)

	return perms
}
