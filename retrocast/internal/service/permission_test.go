package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// Embedding the interfaces makes any unconfigured query panic. These fakes
// implement only queries used by the real checker, with a separate call trace.
type permissionFixture struct {
	t               *testing.T
	guild           *models.Guild
	member          *models.Member
	roles, allRoles []models.Role
	overrides       []models.ChannelOverride
	failAt          string
	calls           []string
}

func fixture(t *testing.T) *permissionFixture {
	return &permissionFixture{
		t:     t,
		guild: &models.Guild{ID: 10, OwnerID: 99}, member: &models.Member{GuildID: 10, UserID: 20},
		roles:    []models.Role{{ID: 2, Permissions: 3, Position: 4}},
		allRoles: []models.Role{{ID: 1, Permissions: 1, IsDefault: true}},
	}
}

func (f *permissionFixture) query(ctx context.Context, name string, guildID, userID int64) error {
	f.t.Helper()
	if ctx.Value(permissionContextKey{}) != "synthetic" || guildID != 10 || (userID != 0 && userID != 20) {
		f.t.Fatalf("wrong query context/IDs for %s: %d %d", name, guildID, userID)
	}
	f.calls = append(f.calls, name)
	if f.failAt == name {
		return errors.New("synthetic repository failure")
	}
	return nil
}

type permissionContextKey struct{}
type permissionGuilds struct {
	database.GuildRepository
	f *permissionFixture
}
type permissionMembers struct {
	database.MemberRepository
	f *permissionFixture
}
type permissionRoles struct {
	database.RoleRepository
	f *permissionFixture
}
type permissionOverrides struct {
	database.ChannelOverrideRepository
	f *permissionFixture
}

func (r permissionGuilds) GetByID(ctx context.Context, id int64) (*models.Guild, error) {
	return r.f.guild, r.f.query(ctx, "guild", id, 0)
}
func (r permissionMembers) GetByGuildAndUser(ctx context.Context, guildID, userID int64) (*models.Member, error) {
	return r.f.member, r.f.query(ctx, "member", guildID, userID)
}
func (r permissionRoles) GetByMember(ctx context.Context, guildID, userID int64) ([]models.Role, error) {
	return r.f.roles, r.f.query(ctx, "roles", guildID, userID)
}
func (r permissionRoles) GetByGuildID(ctx context.Context, guildID int64) ([]models.Role, error) {
	return r.f.allRoles, r.f.query(ctx, "allRoles", guildID, 0)
}
func (r permissionOverrides) GetByChannel(ctx context.Context, channelID int64) ([]models.ChannelOverride, error) {
	if channelID != 30 {
		r.f.t.Fatalf("wrong channel ID: %d", channelID)
	}
	return r.f.overrides, r.f.query(ctx, "overrides", 10, 0)
}

type permissionChecks interface {
	RequireGuildPermission(context.Context, int64, int64, int64) error
	RequireGuildPermissionByPerm(context.Context, int64, int64, permissions.Permission) error
	RequireChannelPermission(context.Context, int64, int64, int64, permissions.Permission) error
}

func checkPermission(checker permissionChecks, ctx context.Context, mode string, mask int64) error {
	switch mode {
	case "raw":
		return checker.RequireGuildPermission(ctx, 10, 20, mask)
	case "typed":
		return checker.RequireGuildPermissionByPerm(ctx, 10, 20, permissions.Permission(mask))
	default:
		return checker.RequireChannelPermission(ctx, 10, 30, 20, permissions.Permission(mask))
	}
}

func errorPolicy(err error) string {
	if err == nil {
		return ""
	}
	var serviceError *ServiceError
	if !errors.As(err, &serviceError) {
		return "unexpected: " + err.Error()
	}
	return serviceError.Code + ":" + serviceError.Message
}

func TestPermissionCheckerFrozenPolicyParity(t *testing.T) {
	// Every row invokes all three public paths on real PermissionChecker and an
	// independent pre-port checker, comparing errors AND repository order/filters.
	cases := []struct {
		name   string
		mask   int64
		change func(*permissionFixture)
	}{
		{"normal", 3, func(*permissionFixture) {}},
		{"owner skips membership and all repository failures", -1, func(f *permissionFixture) { f.guild.OwnerID = 20; f.member = nil; f.failAt = "member" }},
		{"missing guild", 1, func(f *permissionFixture) { f.guild = nil }},
		{"guild query error", 1, func(f *permissionFixture) { f.failAt = "guild" }},
		{"missing member", 1, func(f *permissionFixture) { f.member = nil }},
		{"member query error", 1, func(f *permissionFixture) { f.failAt = "member" }},
		{"member roles error", 1, func(f *permissionFixture) { f.failAt = "roles" }},
		{"guild roles error", 1, func(f *permissionFixture) { f.failAt = "allRoles" }},
		{"override query error", 1, func(f *permissionFixture) { f.failAt = "overrides" }},
		{"admin still queries overrides", 1, func(f *permissionFixture) {
			f.roles[0].Permissions = int64(permissions.PermAdministrator)
			f.failAt = "overrides"
		}},
		{"admin bypasses deny", 3, func(f *permissionFixture) {
			f.roles[0].Permissions = int64(permissions.PermAdministrator)
			f.overrides = []models.ChannelOverride{{RoleID: 1, Deny: -1}, {RoleID: 2, Deny: -1}}
		}},
		{"default role administrator", 3, func(f *permissionFixture) {
			f.roles = nil
			f.allRoles[0].Permissions = int64(permissions.PermAdministrator)
			f.overrides = []models.ChannelOverride{{RoleID: 1, Deny: -1}}
		}},
		{"admin raw bypass differs for sign bit", -(1 << 63), func(f *permissionFixture) { f.roles[0].Permissions = -1 }},
		{"zero requested mask", 0, func(*permissionFixture) {}},
		{"no assigned roles", 1, func(f *permissionFixture) { f.roles = nil }},
		{"no default role", 2, func(f *permissionFixture) { f.allRoles = nil }},
		{"missing default uses zero ID override", 8, func(f *permissionFixture) {
			f.allRoles = nil
			f.overrides = []models.ChannelOverride{{RoleID: 0, Allow: 8}}
		}},
		{"first default role wins", 1, func(f *permissionFixture) {
			f.roles = nil
			f.allRoles = append(f.allRoles, models.Role{ID: 5, IsDefault: true, Permissions: int64(permissions.PermAdministrator)})
		}},
		{"any-bit raw differs from all-bit typed", 3, func(f *permissionFixture) { f.roles[0].Permissions = 1 }},
		{"unknown sign bit survives", -(1 << 63), func(f *permissionFixture) { f.roles[0].Permissions = -(1 << 63) }},
		{"everyone deny", 2, func(f *permissionFixture) { f.overrides = []models.ChannelOverride{{RoleID: 1, Deny: 2}} }},
		{"everyone last override wins", 2, func(f *permissionFixture) {
			f.overrides = []models.ChannelOverride{{RoleID: 1, Deny: 2}, {RoleID: 1, Allow: 2}}
		}},
		{"nonmember overrides ignored", 2, func(f *permissionFixture) {
			f.roles[0].Permissions = 0
			f.overrides = []models.ChannelOverride{{RoleID: 999, Allow: 2}}
		}},
		{"nonmember deny ignored", 2, func(f *permissionFixture) {
			f.overrides = []models.ChannelOverride{{RoleID: 999, Deny: -1}}
		}},
		{"assigned role allow beats cross role deny", 2, func(f *permissionFixture) {
			f.roles = append(f.roles, models.Role{ID: 3})
			f.overrides = []models.ChannelOverride{{RoleID: 1, Deny: 2}, {RoleID: 2, Deny: 2}, {RoleID: 3, Allow: 2}, {RoleID: 3, Allow: 2}}
		}},
		{"everyone ID excluded from assigned overrides", 2, func(f *permissionFixture) {
			f.roles = append(f.roles, models.Role{ID: 1})
			f.overrides = []models.ChannelOverride{{RoleID: 1, Allow: 2}, {RoleID: 1, Deny: 2}}
		}},
		{"negative deny clears all", 1, func(f *permissionFixture) { f.overrides = []models.ChannelOverride{{RoleID: 2, Deny: -1}} }},
		{"negative allow sets all signed bits", -(1 << 63), func(f *permissionFixture) { f.overrides = []models.ChannelOverride{{RoleID: 2, Allow: -1, Deny: -1}} }},
	}
	ctx := context.WithValue(context.Background(), permissionContextKey{}, "synthetic")
	for _, tc := range cases {
		for _, mode := range []string{"raw", "typed", "channel"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				actual, frozen := fixture(t), fixture(t)
				tc.change(actual)
				tc.change(frozen)
				checker := NewPermissionChecker(permissionGuilds{f: actual}, permissionMembers{f: actual}, permissionRoles{f: actual}, permissionOverrides{f: actual})
				oracle := newFrozenPermissionChecker(permissionGuilds{f: frozen}, permissionMembers{f: frozen}, permissionRoles{f: frozen}, permissionOverrides{f: frozen})
				got, want := checkPermission(checker, ctx, mode, tc.mask), checkPermission(oracle, ctx, mode, tc.mask)
				if errorPolicy(got) != errorPolicy(want) || !reflect.DeepEqual(actual.calls, frozen.calls) {
					t.Fatalf("got error=%v calls=%v; want error=%v calls=%v", got, actual.calls, want, frozen.calls)
				}
			})
		}
	}
}

func TestPermissionCheckerDenialsAreErrors(t *testing.T) {
	f := fixture(t)
	f.roles, f.allRoles = nil, nil
	checker := NewPermissionChecker(permissionGuilds{f: f}, permissionMembers{f: f}, permissionRoles{f: f}, permissionOverrides{f: f})
	ctx := context.WithValue(context.Background(), permissionContextKey{}, "synthetic")
	for _, mode := range []string{"raw", "typed", "channel"} {
		if err := checkPermission(checker, ctx, mode, 2); !errors.Is(err, ErrForbidden) {
			t.Fatalf("%s denied access must return ErrForbidden, got %v", mode, err)
		}
	}
	f.member = nil
	if err := checker.RequireGuildPermission(ctx, 10, 20, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("raw missing member: %v", err)
	}
	if err := checker.RequireGuildPermissionByPerm(ctx, 10, 20, 1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("typed missing member: %v", err)
	}
	if err := checker.RequireChannelPermission(ctx, 10, 30, 20, 1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("channel missing member: %v", err)
	}
}

func TestPermissionCheckerMetadataQueries(t *testing.T) {
	f := fixture(t)
	checker := NewPermissionChecker(permissionGuilds{f: f}, permissionMembers{f: f}, permissionRoles{f: f}, permissionOverrides{f: f})
	ctx := context.WithValue(context.Background(), permissionContextKey{}, "synthetic")
	owner, err := checker.IsGuildOwner(ctx, 10, 20)
	if owner || err != nil {
		t.Fatalf("nonowner: %v %v", owner, err)
	}
	f.guild.OwnerID = 20
	owner, err = checker.IsGuildOwner(ctx, 10, 20)
	if !owner || err != nil {
		t.Fatalf("owner: %v %v", owner, err)
	}
	f.guild = nil
	owner, err = checker.IsGuildOwner(ctx, 10, 20)
	if owner || err != nil {
		t.Fatalf("missing guild: %v %v", owner, err)
	}
	f.failAt = "guild"
	owner, err = checker.IsGuildOwner(ctx, 10, 20)
	if owner || !errors.Is(err, ErrInternal) {
		t.Fatalf("query error must be false and internal: %v %v", owner, err)
	}
	f.failAt = ""
	f.roles = append(f.roles, models.Role{Position: 9}, models.Role{Position: -1})
	position, err := checker.HighestRolePosition(ctx, 10, 20)
	if position != 9 || err != nil {
		t.Fatalf("position: %d %v", position, err)
	}
	f.failAt = "roles"
	position, err = checker.HighestRolePosition(ctx, 10, 20)
	if position != 0 || !errors.Is(err, ErrInternal) {
		t.Fatalf("position query error: %d %v", position, err)
	}
}
