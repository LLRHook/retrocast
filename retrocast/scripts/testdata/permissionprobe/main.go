// A synthetic native package probe. It never starts the application or services.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
)

type guilds struct{ database.GuildRepository }
type members struct{ database.MemberRepository }
type roles struct{ database.RoleRepository }
type overrides struct {
	database.ChannelOverrideRepository
}

func (guilds) GetByID(context.Context, int64) (*models.Guild, error) {
	return &models.Guild{ID: 10, OwnerID: 99}, nil
}
func (members) GetByGuildAndUser(context.Context, int64, int64) (*models.Member, error) {
	return &models.Member{GuildID: 10, UserID: 20}, nil
}
func (roles) GetByMember(context.Context, int64, int64) ([]models.Role, error) {
	return []models.Role{{ID: 2, Permissions: 2}}, nil
}
func (roles) GetByGuildID(context.Context, int64) ([]models.Role, error) {
	return []models.Role{{ID: 1, Permissions: 1, IsDefault: true}}, nil
}
func (overrides) GetByChannel(context.Context, int64) ([]models.ChannelOverride, error) {
	return []models.ChannelOverride{{RoleID: 1, Deny: 2}, {RoleID: 2, Allow: 2}, {RoleID: 999, Allow: -1}}, nil
}

func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}

func expectFault(mask permissions.Permission, err error, status int32) {
	var native *permissions.NativeError
	require(mask == 0 && errors.As(err, &native) && native.Status == status,
		fmt.Sprintf("native fault was lost: mask=%d err=%v", mask, err))
}

func legacyMustPanic(call func() permissions.Permission, status int32) {
	defer func() {
		native, ok := recover().(*permissions.NativeError)
		require(ok && native.Status == status, "legacy signature did not panic with native fault")
	}()
	_ = call()
}

func main() {
	checker := service.NewPermissionChecker(guilds{}, members{}, roles{}, overrides{})
	ctx := context.Background()
	if len(os.Args) >= 2 {
		value, err := strconv.ParseInt(os.Args[1], 10, 32)
		require(err == nil && value != 0, "invalid synthetic fault status")
		status := int32(value)
		channelOnly := len(os.Args) == 3 && os.Args[2] == "channel-only"
		mask, err := permissions.ComputeBasePermissionsChecked(models.Role{Permissions: -1}, nil)
		if channelOnly {
			require(mask == 2 && err == nil, "channel-only fault fixture did not reach the channel ABI")
		} else {
			expectFault(mask, err, status)
			legacyMustPanic(func() permissions.Permission { return permissions.ComputeBasePermissions(models.Role{}, nil) }, status)
		}
		mask, err = permissions.ComputeChannelPermissionsChecked(permissions.PermAll, nil, nil)
		expectFault(mask, err, status)
		legacyMustPanic(func() permissions.Permission { return permissions.ComputeChannelPermissions(0, nil, nil) }, status)
		for _, requested := range []permissions.Permission{0, 2, permissions.PermAll, -(1 << 63)} {
			failures := []error{checker.RequireChannelPermission(ctx, 10, 30, 20, requested)}
			if !channelOnly {
				failures = append(failures, checker.RequireGuildPermission(ctx, 10, 20, int64(requested)),
					checker.RequireGuildPermissionByPerm(ctx, 10, 20, requested))
			}
			for _, err := range failures {
				var native *permissions.NativeError
				var public *service.ServiceError
				require(errors.Is(err, service.ErrInternal) && errors.As(err, &native) && native.Status == status,
					fmt.Sprintf("service did not fail closed with native diagnostics: %v", err))
				require(errors.As(err, &public) && public.Code == "INTERNAL" && public.Message == "internal server error",
					"native fault changed public error policy")
			}
		}
		// Owner bypass remains outside the math and never touches the transport.
		require(checker.RequireChannelPermission(ctx, 10, 30, 99, permissions.PermAll) == nil, "owner bypass changed")
		fmt.Printf("PASS synthetic ABI status %d (channel-only=%v): zero masks, legacy panics, service fail-closed and retained diagnostics\n", status, channelOnly)
		return
	}
	base := permissions.ComputeBasePermissions(models.Role{Permissions: -(1 << 63)}, []models.Role{{Permissions: 3}})
	require(base == -(1<<63)|3, "packaged base resolver lost signed bits")
	channel := permissions.ComputeChannelPermissions(base, &models.ChannelOverride{Deny: -1}, []models.ChannelOverride{{Allow: -(1 << 63), Deny: -1}, {Allow: 2}})
	require(channel == -(1<<63)|2, "packaged channel resolver changed override precedence")
	require(checker.RequireGuildPermission(ctx, 10, 20, 2) == nil, "raw guild path failed")
	require(checker.RequireGuildPermissionByPerm(ctx, 10, 20, 3) == nil, "typed guild path failed")
	require(checker.RequireChannelPermission(ctx, 10, 30, 20, 3) == nil, "channel path failed")
	require(errors.Is(checker.RequireChannelPermission(ctx, 10, 30, 20, 8), service.ErrForbidden), "unassigned role granted access")
	fmt.Println("PASS relocated native package: signed masks, override precedence, real guild/channel checker and denied access")
}
