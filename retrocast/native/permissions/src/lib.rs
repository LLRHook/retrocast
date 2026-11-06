//! Complete base/channel permission resolution. No role metadata or repository policy.

use std::mem::{align_of, size_of};
use std::panic::{UnwindSafe, catch_unwind};
use std::slice;

const ABI_VERSION: u32 = 1;
const ADMINISTRATOR: i64 = 1 << 31;
const ALL: i64 = i64::MAX;
const OK: i32 = 0;
const ABI_MISMATCH: i32 = 1;
const INVALID_ARGUMENT: i32 = 2;
const PANIC: i32 = 3;

#[repr(C)]
#[derive(Clone, Copy, Debug)]
pub struct Override {
    pub allow: i64,
    pub deny: i64,
}

#[repr(C)]
#[derive(Debug, PartialEq, Eq)]
pub struct PermissionResult {
    pub permissions: i64,
    pub status: i32,
}

impl PermissionResult {
    fn error(status: i32) -> Self {
        Self {
            permissions: 0,
            status,
        }
    }
}

fn base_permissions(everyone: i64, roles: &[i64]) -> i64 {
    let permissions = roles.iter().fold(everyone, |p, role| p | role);
    if permissions & ADMINISTRATOR != 0 {
        ALL
    } else {
        permissions
    }
}

fn channel_permissions(base: i64, everyone: Option<Override>, overrides: &[Override]) -> i64 {
    // Only the incoming base triggers bypass. An override that adds ADMINISTRATOR
    // does not retroactively bypass later overrides in the legacy algorithm.
    if base & ADMINISTRATOR != 0 {
        return ALL;
    }
    let mut permissions = base;
    if let Some(everyone) = everyone {
        permissions = (permissions & !everyone.deny) | everyone.allow;
    }
    let (allow, deny) = overrides
        .iter()
        .fold((0, 0), |(allow, deny), o| (allow | o.allow, deny | o.deny));
    (permissions & !deny) | allow
}

fn guarded(f: impl FnOnce() -> i64 + UnwindSafe) -> PermissionResult {
    match catch_unwind(f) {
        Ok(permissions) => PermissionResult {
            permissions,
            status: OK,
        },
        Err(_) => PermissionResult::error(PANIC),
    }
}

fn valid_array<T>(ptr: *const T, len: usize) -> bool {
    len == 0
        || (!ptr.is_null()
            && (ptr as usize).is_multiple_of(align_of::<T>())
            && len <= isize::MAX as usize / size_of::<T>())
}

// The caller owns pointer validity. These checks reject representational faults;
// they cannot validate an arbitrary address supplied by a corrupt foreign caller.
unsafe fn borrowed<'a, T>(ptr: *const T, len: usize) -> &'a [T] {
    if len == 0 {
        &[]
    } else {
        // SAFETY: caller supplies a valid, aligned, initialized array for this call.
        unsafe { slice::from_raw_parts(ptr, len) }
    }
}

/// Resolve the entire base mask, including Administrator normalization.
///
/// # Safety
/// For nonzero roles_len, roles must reference that many initialized i64 values
/// in one allocation, aligned and live without mutation until the call returns.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn rc_permissions_base(
    abi_version: u32,
    everyone: i64,
    roles: *const i64,
    roles_len: usize,
) -> PermissionResult {
    if abi_version != ABI_VERSION {
        return PermissionResult::error(ABI_MISMATCH);
    }
    if !valid_array(roles, roles_len) {
        return PermissionResult::error(INVALID_ARGUMENT);
    }
    guarded(|| {
        // SAFETY: ABI preconditions and representational checks above.
        base_permissions(everyone, unsafe { borrowed(roles, roles_len) })
    })
}

/// Resolve the entire channel mask, including aggregated role overrides.
///
/// # Safety
/// For nonzero overrides_len, overrides must reference that many initialized
/// Override values in one aligned allocation, live and immutable for this call.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn rc_permissions_channel(
    abi_version: u32,
    base: i64,
    has_everyone: u32,
    everyone_allow: i64,
    everyone_deny: i64,
    overrides: *const Override,
    overrides_len: usize,
) -> PermissionResult {
    if abi_version != ABI_VERSION {
        return PermissionResult::error(ABI_MISMATCH);
    }
    if has_everyone > 1 || !valid_array(overrides, overrides_len) {
        return PermissionResult::error(INVALID_ARGUMENT);
    }
    guarded(|| {
        let everyone = (has_everyone == 1).then_some(Override {
            allow: everyone_allow,
            deny: everyone_deny,
        });
        // SAFETY: ABI preconditions and representational checks above.
        channel_permissions(base, everyone, unsafe {
            borrowed(overrides, overrides_len)
        })
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::ptr;

    #[test]
    fn base_keeps_signed_bits_and_normalizes_only_administrators() {
        assert_eq!(
            base_permissions(i64::MIN, &[1 << 62, 3, 3]),
            i64::MIN | (1 << 62) | 3
        );
        assert_eq!(base_permissions(i64::MIN, &[ADMINISTRATOR]), ALL);
        assert_eq!(base_permissions(-1, &[]), ALL);
    }

    #[test]
    fn channel_order_conflicts_and_admin_timing() {
        let everyone = Override { allow: 3, deny: -1 };
        let overrides = [
            Override { allow: 4, deny: 3 },
            Override { allow: 2, deny: 4 },
        ];
        assert_eq!(channel_permissions(1, Some(everyone), &overrides), 6);
        assert_eq!(
            channel_permissions(ADMINISTRATOR | i64::MIN, Some(everyone), &overrides),
            ALL
        );
        assert_eq!(
            channel_permissions(
                0,
                Some(Override {
                    allow: ADMINISTRATOR,
                    deny: 0
                }),
                &[Override {
                    allow: 0,
                    deny: ADMINISTRATOR
                }]
            ),
            0
        );
    }

    #[test]
    fn null_empty_arrays_are_valid() {
        unsafe {
            assert_eq!(
                rc_permissions_base(ABI_VERSION, i64::MIN, ptr::null(), 0),
                PermissionResult {
                    permissions: i64::MIN,
                    status: OK
                }
            );
            assert_eq!(
                rc_permissions_channel(ABI_VERSION, 7, 0, -1, -1, ptr::null(), 0),
                PermissionResult {
                    permissions: 7,
                    status: OK
                }
            );
        }
    }

    #[test]
    fn invalid_abi_is_rejected_before_arguments() {
        unsafe {
            assert_eq!(
                rc_permissions_base(0, -1, ptr::null(), 1),
                PermissionResult::error(ABI_MISMATCH)
            );
            assert_eq!(
                rc_permissions_channel(0, -1, 9, 0, 0, ptr::null(), 1),
                PermissionResult::error(ABI_MISMATCH)
            );
        }
    }

    #[test]
    fn malformed_arrays_and_flags_fail_closed_even_for_admin() {
        let role = 0_i64;
        let override_value = Override { allow: 0, deny: 0 };
        unsafe {
            assert_eq!(
                rc_permissions_base(ABI_VERSION, ADMINISTRATOR, ptr::null(), 1),
                PermissionResult::error(INVALID_ARGUMENT)
            );
            assert_eq!(
                rc_permissions_base(ABI_VERSION, 0, &role, usize::MAX),
                PermissionResult::error(INVALID_ARGUMENT)
            );
            let unaligned = (&role as *const i64)
                .cast::<u8>()
                .wrapping_add(1)
                .cast::<i64>();
            assert_eq!(
                rc_permissions_base(ABI_VERSION, 0, unaligned, 1),
                PermissionResult::error(INVALID_ARGUMENT)
            );
            for (flag, ptr, len) in [
                (2, &override_value as *const Override, 0),
                (1, ptr::null(), 1),
                (0, &override_value, usize::MAX),
            ] {
                assert_eq!(
                    rc_permissions_channel(ABI_VERSION, ADMINISTRATOR, flag, 0, 0, ptr, len),
                    PermissionResult::error(INVALID_ARGUMENT)
                );
            }
        }
    }

    #[test]
    fn panic_is_a_zero_mask_error() {
        assert_eq!(
            guarded(|| panic!("synthetic transport fault")),
            PermissionResult::error(PANIC)
        );
    }

    #[test]
    fn ffi_borrows_without_mutation_or_retention() {
        for _ in 0..1000 {
            // Heap allocations are dropped/recreated between calls to exercise
            // the ABI's borrowed lifetime rather than a static test array.
            let roles = Box::new([i64::MIN, 5, 5]);
            let overrides = Box::new([Override {
                allow: i64::MIN,
                deny: -1,
            }]);
            unsafe {
                assert_eq!(
                    rc_permissions_base(ABI_VERSION, 2, roles.as_ptr(), roles.len()).permissions,
                    i64::MIN | 7
                );
                assert_eq!(
                    rc_permissions_channel(ABI_VERSION, 7, 0, 0, 0, overrides.as_ptr(), 1)
                        .permissions,
                    i64::MIN
                );
            }
            assert_eq!(*roles, [i64::MIN, 5, 5]);
            assert_eq!(overrides[0].deny, -1);
        }
    }
}
