package permissions

import (
	"errors"
	"math/rand"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/victorivanov/retrocast/internal/models"
)

func assertBaseParity(t *testing.T, everyone models.Role, roles []models.Role) {
	t.Helper()
	want := frozenBasePermissions(everyone, roles)
	got, err := ComputeBasePermissionsChecked(everyone, roles)
	if err != nil || got != want {
		t.Fatalf("base everyone=%#x roles=%v: got %#x, %v; want %#x", everyone.Permissions, roles, int64(got), err, int64(want))
	}
	if legacy := ComputeBasePermissions(everyone, roles); legacy != want {
		t.Fatalf("public base wrapper got %#x, want %#x", int64(legacy), int64(want))
	}
}

func assertChannelParity(t *testing.T, base Permission, everyone *models.ChannelOverride, roles []models.ChannelOverride) {
	t.Helper()
	want := frozenChannelPermissions(base, everyone, roles)
	got, err := ComputeChannelPermissionsChecked(base, everyone, roles)
	if err != nil || got != want {
		t.Fatalf("channel base=%#x everyone=%+v roles=%v: got %#x, %v; want %#x", int64(base), everyone, roles, int64(got), err, int64(want))
	}
	if legacy := ComputeChannelPermissions(base, everyone, roles); legacy != want {
		t.Fatalf("public channel wrapper got %#x, want %#x", int64(legacy), int64(want))
	}
}

func TestRustBaseExhaustive(t *testing.T) {
	// 16^3 = 4096 unions; metadata is deliberately irrelevant to the engine.
	for everyone := int64(0); everyone < 16; everyone++ {
		for first := int64(0); first < 16; first++ {
			for second := int64(0); second < 16; second++ {
				assertBaseParity(t, models.Role{Permissions: everyone}, []models.Role{
					{ID: 2, Name: "synthetic", Permissions: first}, {ID: 2, Permissions: second},
				})
			}
		}
	}
}

func TestRustChannelExhaustiveConflicts(t *testing.T) {
	// 4^7 masks, each with and without everyone: 32768 complete resolutions.
	// The two roles independently allow/deny every bounded mask, including all
	// cross-role conflicts. Duplicate IDs are intentional: the math ignores IDs.
	for encoded := 0; encoded < 1<<14; encoded++ {
		n := encoded
		next := func() int64 { value := int64(n & 3); n >>= 2; return value }
		base := Permission(next())
		everyone := &models.ChannelOverride{Allow: next(), Deny: next()}
		roles := []models.ChannelOverride{
			{RoleID: 4, Allow: next(), Deny: next()}, {RoleID: 4, Allow: next(), Deny: next()},
		}
		assertChannelParity(t, base, everyone, roles)
		assertChannelParity(t, base, nil, roles)
	}
}

func TestRustSignedEdgesNilDuplicatesAndOrder(t *testing.T) {
	masks := []int64{0, 1, -1, -2, -(1 << 63), 1 << 62, int64(PermAdministrator), int64(PermAll), -(1 << 63) | 5}
	for _, mask := range masks {
		assertBaseParity(t, models.Role{Permissions: mask}, nil)
		assertBaseParity(t, models.Role{Permissions: mask}, []models.Role{})
		assertChannelParity(t, Permission(mask), nil, nil)
		assertChannelParity(t, Permission(mask), &models.ChannelOverride{}, []models.ChannelOverride{})
		for _, other := range masks {
			roles := []models.Role{{Permissions: mask}, {Permissions: other}, {Permissions: mask}}
			assertBaseParity(t, models.Role{}, roles)
			overrides := []models.ChannelOverride{{Allow: mask, Deny: other}, {Allow: other, Deny: mask}}
			before := ComputeChannelPermissions(Permission(mask), nil, overrides)
			assertChannelParity(t, Permission(mask), &models.ChannelOverride{Allow: other, Deny: mask}, overrides)
			overrides[0], overrides[1] = overrides[1], overrides[0]
			overrides = append(overrides, overrides[0])
			if after := ComputeChannelPermissions(Permission(mask), nil, overrides); before != after {
				t.Fatalf("order/duplicate changed result: %#x -> %#x", int64(before), int64(after))
			}
		}
	}
	// An override may add Administrator; only the incoming base bypasses.
	assertChannelParity(t, 0, &models.ChannelOverride{Allow: int64(PermAdministrator)},
		[]models.ChannelOverride{{Deny: int64(PermAdministrator), Allow: -(1 << 63)}})
}

func TestRustSeededFull64BitDifferential(t *testing.T) {
	rng := rand.New(rand.NewSource(0x726574726f))
	for sample := 0; sample < 10000; sample++ {
		everyone := models.Role{Permissions: int64(rng.Uint64())}
		roles := make([]models.Role, rng.Intn(12))
		overrides := make([]models.ChannelOverride, rng.Intn(12))
		for i := range roles {
			roles[i] = models.Role{ID: int64(i % 3), Permissions: int64(rng.Uint64())}
		}
		for i := range overrides {
			overrides[i] = models.ChannelOverride{RoleID: int64(i % 3), Allow: int64(rng.Uint64()), Deny: int64(rng.Uint64())}
		}
		// Exercise both admin and non-admin full-width masks, rather than letting
		// large random role unions almost always collapse to PermAll.
		if sample%2 == 0 {
			everyone.Permissions &^= int64(PermAdministrator)
			for i := range roles {
				roles[i].Permissions &^= int64(PermAdministrator)
			}
		}
		assertBaseParity(t, everyone, roles)
		base := Permission(int64(rng.Uint64()))
		if sample%2 == 0 {
			base &^= PermAdministrator
		}
		var everyoneOverride *models.ChannelOverride
		if sample%3 != 0 {
			everyoneOverride = &models.ChannelOverride{Allow: int64(rng.Uint64()), Deny: int64(rng.Uint64())}
		}
		assertChannelParity(t, base, everyoneOverride, overrides)
		assertChannelParity(t, ComputeBasePermissions(everyone, roles), everyoneOverride, overrides)
	}
}

func TestRustConcurrentBorrowedLifetimes(t *testing.T) {
	var workers sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		workers.Add(1)
		go func(seed int64) {
			defer workers.Done()
			rng := rand.New(rand.NewSource(seed))
			for iteration := 0; iteration < 1000; iteration++ {
				roles := []models.Role{{Name: "Go-owned", Permissions: int64(rng.Uint64()) &^ int64(PermAdministrator)}}
				overrides := []models.ChannelOverride{{Allow: int64(rng.Uint64()), Deny: int64(rng.Uint64())}}
				roleCopy, overrideCopy := append([]models.Role(nil), roles...), append([]models.ChannelOverride(nil), overrides...)
				assertBaseParity(t, models.Role{}, roles)
				assertChannelParity(t, Permission(roles[0].Permissions), nil, overrides)
				if !reflect.DeepEqual(roles, roleCopy) || !reflect.DeepEqual(overrides, overrideCopy) {
					t.Error("Rust changed caller-owned input")
					return
				}
				if iteration%200 == 0 {
					runtime.GC()
				}
			}
		}(int64(worker))
	}
	workers.Wait()
}

func TestRustABIMismatchFailsClosed(t *testing.T) {
	for _, version := range []uint32{0, nativeABIVersion + 1, ^uint32(0)} {
		mask, err := nativeBase(version, int64(PermAll), nil)
		assertNativeFailure(t, mask, err, "base", 1)
		mask, err = nativeChannel(version, PermAll, nil, nil)
		assertNativeFailure(t, mask, err, "channel", 1)
	}
}

func assertNativeFailure(t *testing.T, mask Permission, err error, operation string, status int32) {
	t.Helper()
	var native *NativeError
	if mask != 0 || !errors.As(err, &native) || native.Operation != operation || native.Status != status {
		t.Fatalf("got mask=%#x err=%v; want zero mask with %s status %d", int64(mask), err, operation, status)
	}
}

func TestNativeErrorNeverBecomesPermissions(t *testing.T) {
	for _, status := range []int32{1, 2, 3, -1, 99} {
		mask, err := nativeResult("base", int64(PermAll), status)
		assertNativeFailure(t, mask, err, "base", status)
		func() {
			defer func() {
				recovered := recover()
				if recovered != err {
					t.Errorf("legacy wrapper must panic with the native error, got %v", recovered)
				}
			}()
			_ = mustPermission(mask, err)
		}()
	}
}

func TestNativeErrorIncludesArchiveIdentity(t *testing.T) {
	_, err := nativeResult("base", int64(PermAll), 99)
	if len(nativeArchiveFingerprint) != 64 || !strings.Contains(err.Error(), "native build "+nativeArchiveFingerprint) {
		t.Fatalf("native archive identity missing from observable error: %v", err)
	}
}
