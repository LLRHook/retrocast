/* Deliberately hostile ABI fixture, linked only in an isolated test module.
 * Returning a grant mask with an error verifies that hosts discard that mask.
 */
#include "permissions.h"
#ifndef RC_TEST_STATUS
#error RC_TEST_STATUS is required
#endif

rc_permission_result rc_permissions_base(uint32_t version, int64_t everyone,
    const int64_t *roles, size_t len) {
    (void)version; (void)everyone; (void)roles; (void)len;
#if RC_TEST_BASE_OK
    return (rc_permission_result){2, RC_PERMISSIONS_OK};
#else
    return (rc_permission_result){INT64_MAX, RC_TEST_STATUS};
#endif
}

rc_permission_result rc_permissions_channel(uint32_t version, int64_t base,
    uint32_t present, int64_t allow, int64_t deny,
    const rc_permission_override *overrides, size_t len) {
    (void)version; (void)base; (void)present; (void)allow; (void)deny;
    (void)overrides; (void)len;
    return (rc_permission_result){INT64_MAX, RC_TEST_STATUS};
}
