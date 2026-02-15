package permissions

import (
	"testing"

	"github.com/victorivanov/retrocast/internal/models"
)

func TestComputeBasePermissions_EveryoneOnly(t *testing.T) {
	everyone := models.Role{
		Permissions: int64(PermViewChannel | PermSendMessages),
		IsDefault:   true,
	}
	perms := ComputeBasePermissions(everyone, nil)
	if !perms.Has(PermViewChannel | PermSendMessages) {
		t.Error("expected base perms to include @everyone permissions")
	}
	if perms.Has(PermManageMessages) {
		t.Error("expected ManageMessages to not be set")
	}
}

func TestComputeBasePermissions_WithRoles(t *testing.T) {
	everyone := models.Role{
		Permissions: int64(PermViewChannel),
		IsDefault:   true,
	}
	roles := []models.Role{
		{Permissions: int64(PermSendMessages)},
		{Permissions: int64(PermManageMessages)},
	}
	perms := ComputeBasePermissions(everyone, roles)
	if !perms.Has(PermViewChannel | PermSendMessages | PermManageMessages) {
		t.Error("expected perms to combine @everyone and assigned roles")
	}
}

func TestComputeBasePermissions_AdministratorBypass(t *testing.T) {
	everyone := models.Role{
		Permissions: int64(PermViewChannel),
		IsDefault:   true,
	}
	roles := []models.Role{
		{Permissions: int64(PermAdministrator)},
	}
	perms := ComputeBasePermissions(everyone, roles)
	if perms != PermAll {
		t.Errorf("expected PermAll when Administrator is set, got %d", perms)
	}
}

func TestComputeBasePermissions_AdministratorOnEveryone(t *testing.T) {
	everyone := models.Role{
		Permissions: int64(PermAdministrator),
		IsDefault:   true,
	}
	perms := ComputeBasePermissions(everyone, nil)
	if perms != PermAll {
		t.Error("Administrator on @everyone should grant PermAll")
	}
}

func TestComputeChannelPermissions_NoOverrides(t *testing.T) {
	base := PermViewChannel | PermSendMessages
	perms := ComputeChannelPermissions(base, nil, nil)
	if perms != base {
		t.Error("with no overrides, channel perms should equal base perms")
	}
}

func TestComputeChannelPermissions_AdministratorBypass(t *testing.T) {
	base := PermAdministrator | PermViewChannel
	everyone := &models.ChannelOverride{
		Deny: int64(PermViewChannel),
	}
	perms := ComputeChannelPermissions(base, everyone, nil)
	if perms != PermAll {
		t.Error("Administrator should bypass channel overrides and return PermAll")
	}
}

func TestComputeChannelPermissions_EveryoneOverrideDeny(t *testing.T) {
	base := PermViewChannel | PermSendMessages
	everyone := &models.ChannelOverride{
		Deny: int64(PermSendMessages),
	}
	perms := ComputeChannelPermissions(base, everyone, nil)
	if perms.Has(PermSendMessages) {
		t.Error("@everyone deny should remove SendMessages")
	}
	if !perms.Has(PermViewChannel) {
		t.Error("ViewChannel should remain")
	}
}

func TestComputeChannelPermissions_EveryoneOverrideAllow(t *testing.T) {
	base := PermViewChannel
	everyone := &models.ChannelOverride{
		Allow: int64(PermManageMessages),
	}
	perms := ComputeChannelPermissions(base, everyone, nil)
	if !perms.Has(PermManageMessages) {
		t.Error("@everyone allow should add ManageMessages")
	}
}

func TestComputeChannelPermissions_EveryoneOverrideDenyThenAllow(t *testing.T) {
	base := PermViewChannel | PermSendMessages
	// Deny SendMessages but also allow it — allow takes precedence since
	// deny is applied first and then allow is OR-ed on top.
	everyone := &models.ChannelOverride{
		Deny:  int64(PermSendMessages),
		Allow: int64(PermSendMessages),
	}
	perms := ComputeChannelPermissions(base, everyone, nil)
	if !perms.Has(PermSendMessages) {
		t.Error("allow should override deny for @everyone (allow applied after deny)")
	}
}

func TestComputeChannelPermissions_RoleOverrideDeny(t *testing.T) {
	base := PermViewChannel | PermSendMessages | PermManageMessages
	roleOverrides := []models.ChannelOverride{
		{RoleID: 1, Deny: int64(PermManageMessages)},
	}
	perms := ComputeChannelPermissions(base, nil, roleOverrides)
	if perms.Has(PermManageMessages) {
		t.Error("role deny should remove ManageMessages")
	}
	if !perms.Has(PermViewChannel | PermSendMessages) {
		t.Error("other permissions should remain")
	}
}

func TestComputeChannelPermissions_RoleOverrideAllow(t *testing.T) {
	base := PermViewChannel
	roleOverrides := []models.ChannelOverride{
		{RoleID: 1, Allow: int64(PermAttachFiles)},
	}
	perms := ComputeChannelPermissions(base, nil, roleOverrides)
	if !perms.Has(PermAttachFiles) {
		t.Error("role allow should add AttachFiles")
	}
}

func TestComputeChannelPermissions_MultipleRoleOverrides(t *testing.T) {
	base := PermViewChannel | PermSendMessages | PermManageMessages
	roleOverrides := []models.ChannelOverride{
		{RoleID: 1, Allow: int64(PermAttachFiles), Deny: int64(PermManageMessages)},
		{RoleID: 2, Allow: int64(PermMentionEveryone)},
	}
	perms := ComputeChannelPermissions(base, nil, roleOverrides)
	if perms.Has(PermManageMessages) {
		t.Error("deny from any role override should remove ManageMessages")
	}
	if !perms.Has(PermAttachFiles) {
		t.Error("allow from role 1 should add AttachFiles")
	}
	if !perms.Has(PermMentionEveryone) {
		t.Error("allow from role 2 should add MentionEveryone")
	}
}

func TestComputeChannelPermissions_RoleAllowOverridesDeny(t *testing.T) {
	// If one role denies and another allows the same permission,
	// the allow wins because allows are applied after denies.
	base := PermViewChannel | PermSendMessages
	roleOverrides := []models.ChannelOverride{
		{RoleID: 1, Deny: int64(PermSendMessages)},
		{RoleID: 2, Allow: int64(PermSendMessages)},
	}
	perms := ComputeChannelPermissions(base, nil, roleOverrides)
	if !perms.Has(PermSendMessages) {
		t.Error("role allow should override role deny (allow applied after deny)")
	}
}

func TestComputeChannelPermissions_EveryoneBeforeRoles(t *testing.T) {
	// @everyone override is applied first, then role overrides.
	// @everyone denies SendMessages, but a role allows it back.
	base := PermViewChannel | PermSendMessages
	everyone := &models.ChannelOverride{
		Deny: int64(PermSendMessages),
	}
	roleOverrides := []models.ChannelOverride{
		{RoleID: 1, Allow: int64(PermSendMessages)},
	}
	perms := ComputeChannelPermissions(base, everyone, roleOverrides)
	if !perms.Has(PermSendMessages) {
		t.Error("role allow should restore permission denied by @everyone")
	}
}

func TestComputeChannelPermissions_FullScenario(t *testing.T) {
	// Base: view, send, history, connect, speak
	base := PermViewChannel | PermSendMessages | PermReadMessageHistory | PermConnect | PermSpeak

	// @everyone: deny speak
	everyone := &models.ChannelOverride{
		Deny: int64(PermSpeak),
	}

	// Role overrides: role 1 allows manage messages, role 2 denies send messages
	roleOverrides := []models.ChannelOverride{
		{RoleID: 1, Allow: int64(PermManageMessages)},
		{RoleID: 2, Deny: int64(PermSendMessages)},
	}

	perms := ComputeChannelPermissions(base, everyone, roleOverrides)

	if !perms.Has(PermViewChannel) {
		t.Error("ViewChannel should remain")
	}
	if perms.Has(PermSendMessages) {
		t.Error("SendMessages should be denied by role 2")
	}
	if perms.Has(PermSpeak) {
		t.Error("Speak should be denied by @everyone")
	}
	if !perms.Has(PermManageMessages) {
		t.Error("ManageMessages should be allowed by role 1")
	}
	if !perms.Has(PermReadMessageHistory) {
		t.Error("ReadMessageHistory should remain")
	}
	if !perms.Has(PermConnect) {
		t.Error("Connect should remain")
	}
}
