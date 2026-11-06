#ifndef RETROCAST_PERMISSIONS_H
#define RETROCAST_PERMISSIONS_H

#include <stddef.h>
#include <stdint.h>

#define RC_PERMISSIONS_ABI_VERSION 1
#define RC_PERMISSIONS_OK 0
#define RC_PERMISSIONS_ABI_MISMATCH 1
#define RC_PERMISSIONS_INVALID_ARGUMENT 2
#define RC_PERMISSIONS_PANIC 3

typedef struct {
    int64_t allow;
    int64_t deny;
} rc_permission_override;

typedef struct {
    int64_t permissions;
    int32_t status;
} rc_permission_result;

/* Arrays are borrowed only for the call; NULL is valid only for length zero.
 * Nonempty arrays must be aligned, initialized, and valid for their full length.
 * Inputs and results retain all 64 signed bits. An error result has no permissions.
 */
rc_permission_result rc_permissions_base(uint32_t abi_version, int64_t everyone,
    const int64_t *roles, size_t roles_len);
rc_permission_result rc_permissions_channel(uint32_t abi_version, int64_t base,
    uint32_t has_everyone, int64_t everyone_allow, int64_t everyone_deny,
    const rc_permission_override *overrides, size_t overrides_len);

#endif
