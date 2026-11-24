# Pilot acceptance test index

L40 extends the L38 clause map from 26ebe80 on baseline 49ae8dd. Clause numbers below are those in L38. These CI tests supply assertion evidence; Delivery is unchanged and candidate verification belongs to validation run 3.

Run Go tests with Postgres 17 migrations 000001–000020 and Redis 7, using `go test -race -count=1 -v ./...` from `retrocast/`. Run web tests with `npx -y pnpm@10 test` from `retrocast-web/`.

| Requirement | Clause | Asserting test | File |
|---|---|---|---|
| SV-FR-006 | 1 | `TestPilotLogoutRevocation` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-006 | 2 | `TestPilotLogoutRevocation` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-006 | 3 | `TestSessionTransportLive/logout` | [retrocast/internal/api/session_transport_live_test.go](../../retrocast/internal/api/session_transport_live_test.go) |
| SV-FR-006 | 4 | `TestSessionTransportLive/logout` | [retrocast/internal/api/session_transport_live_test.go](../../retrocast/internal/api/session_transport_live_test.go) |
| SV-FR-060 | 4 | `TestRegistrationAtomicityLive/password_session_revocation` | [retrocast/internal/api/atomicity_live_test.go](../../retrocast/internal/api/atomicity_live_test.go) |
| SV-FR-007 | 2 | `TestCreateGuild_Success` | [retrocast/internal/api/guild_handler_test.go](../../retrocast/internal/api/guild_handler_test.go) |
| SV-FR-007 | 3 | `TestCreateGuild_Success` | [retrocast/internal/api/guild_handler_test.go](../../retrocast/internal/api/guild_handler_test.go) |
| SV-FR-009 | 1 | `TestRegistrationAtomicityLive/pilot_defaults_member_identity_and_page` | [retrocast/internal/api/atomicity_live_test.go](../../retrocast/internal/api/atomicity_live_test.go) |
| SV-FR-011 | 3 | `TestUpdateGuild_WithPermission` | [retrocast/internal/api/guild_handler_test.go](../../retrocast/internal/api/guild_handler_test.go) |
| SV-FR-012 | 1 | `TestPilotMissingManagementPermissions/deleteGuild` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-012 | 4 | `TestDeleteGuild_AsOwner` | [retrocast/internal/api/guild_handler_test.go](../../retrocast/internal/api/guild_handler_test.go) |
| SV-FR-013 | 1 | `TestPilotListTwoGuilds` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-014 | 3 | `TestCreateChannel_Success` | [retrocast/internal/api/channel_handler_test.go](../../retrocast/internal/api/channel_handler_test.go) |
| SV-FR-015 | 2 | `TestUpdateChannel_WithPermission` | [retrocast/internal/api/channel_handler_test.go](../../retrocast/internal/api/channel_handler_test.go) |
| SV-FR-015 | 4 | `TestChannelRepo_Delete` | [retrocast/internal/database/channel_repo_test.go](../../retrocast/internal/database/channel_repo_test.go) |
| SV-FR-016 | 1 | `TestPilotNonmemberReads/listChannels` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-016 | 2 | `TestPilotNonmemberReads/getChannel` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-018 | 1 | `TestPilotInviteDefaults` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-018 | 2 | `TestPilotInviteDefaults` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-018 | 3 | `TestPilotInviteDefaults` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-018 | 4 | `TestPilotInviteDefaults` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-019 | 4 | `TestGetInvite_Public` | [retrocast/internal/api/invite_handler_test.go](../../retrocast/internal/api/invite_handler_test.go) |
| SV-FR-019 | 5 | `TestGetInvite_Expired` | [retrocast/internal/api/invite_handler_test.go](../../retrocast/internal/api/invite_handler_test.go) |
| SV-FR-020 | 1 | `TestAcceptInvite_Success` | [retrocast/internal/api/invite_handler_test.go](../../retrocast/internal/api/invite_handler_test.go) |
| SV-FR-020 | 3 | `TestAcceptInvite_Success` | [retrocast/internal/api/invite_handler_test.go](../../retrocast/internal/api/invite_handler_test.go) |
| SV-FR-020 | 4 | `TestPilotInviteRejections/expired` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-020 | 7 | `TestPilotInviteRejections/banned` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-021 | 1 | `TestPilotMissingManagementPermissions/listInvites` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-022 | 3 | `TestLeaveGuild_AsOwner` | [retrocast/internal/api/member_handler_test.go](../../retrocast/internal/api/member_handler_test.go) |
| SV-FR-023 | 1 | `TestRegistrationAtomicityLive/pilot_defaults_member_identity_and_page` | [retrocast/internal/api/atomicity_live_test.go](../../retrocast/internal/api/atomicity_live_test.go) |
| SV-FR-023 | 2 | `TestRegistrationAtomicityLive/pilot_defaults_member_identity_and_page` | [retrocast/internal/api/atomicity_live_test.go](../../retrocast/internal/api/atomicity_live_test.go) |
| SV-FR-024 | 1 | `TestPilotNonmemberReads/getMember` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-025 | 3 | `TestSendMessage_Success` | [retrocast/internal/api/message_handler_test.go](../../retrocast/internal/api/message_handler_test.go) |
| SV-FR-033 | 1 | `TestPilotInvisibleBroadcastsOffline` | [retrocast/internal/gateway/session_live_test.go](../../retrocast/internal/gateway/session_live_test.go) |
| SV-FR-041 | 1 | `TestPermissionRegressionRoleBoundaries/update/foreign` | [retrocast/internal/api/permission_regression_test.go](../../retrocast/internal/api/permission_regression_test.go) |
| SV-FR-041 | 2 | `TestPermissionRegressionRoleOperations/update/400` | [retrocast/internal/api/permission_regression_test.go](../../retrocast/internal/api/permission_regression_test.go) |
| SV-FR-044 | 1 | `TestPermissionRegressionRoleOperations/list/500` | [retrocast/internal/api/permission_regression_test.go](../../retrocast/internal/api/permission_regression_test.go) |
| SV-FR-078 | 11 | `TestEveryonePositionImmutable` | [retrocast/internal/api/permission_round2_test.go](../../retrocast/internal/api/permission_round2_test.go) |
| SV-FR-066 | 2 | `TestModerationAtomicityLive` | [retrocast/internal/api/moderation_live_test.go](../../retrocast/internal/api/moderation_live_test.go) |
| SV-FR-066 | 5 | `TestModerationAtomicityLive` | [retrocast/internal/api/moderation_live_test.go](../../retrocast/internal/api/moderation_live_test.go) |
| SV-FR-067 | 5 | `TestModerationAtomicityLive` | [retrocast/internal/api/moderation_live_test.go](../../retrocast/internal/api/moderation_live_test.go) |
| SV-FR-068 | 2 | `TestListBans` | [retrocast/internal/api/ban_handler_test.go](../../retrocast/internal/api/ban_handler_test.go) |
| SV-FR-068 | 3 | `TestPilotMissingManagementPermissions/listBans` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-069 | 3 | `TestModerationAtomicityLive` | [retrocast/internal/api/moderation_live_test.go](../../retrocast/internal/api/moderation_live_test.go) |
| SV-FR-069 | 4 | `TestPilotMissingManagementPermissions/unban` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-070 | 1 | `TestPilotModerationBoundaries/kick/equal` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-070 | 2 | `TestPilotModerationBoundaries/ban/equal` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-070 | 3 | `TestPilotModerationBoundaries/kick/owner` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-070 | 5 | `TestPilotModerationBoundaries/kick/self` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-070 | 7 | `TestPilotModerationBoundaries/kick/everyone` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-FR-070 | 8 | `TestPilotModerationBoundaries/ban/everyone` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-IR-002 | 1 | `TestLiveMultipleConnectionsAndConcurrentDelivery` | [retrocast/internal/gateway/session_live_test.go](../../retrocast/internal/gateway/session_live_test.go) |
| SV-IR-007 | 1 | `TestContractDocumentedSuccess` | [retrocast/internal/api/contract_test.go](../../retrocast/internal/api/contract_test.go) |
| SV-IR-007 | 2 | `TestContractDocumentedSuccess` | [retrocast/internal/api/contract_test.go](../../retrocast/internal/api/contract_test.go) |
| SV-IR-008 | 2 | `TestContractOpenAPI` | [retrocast/internal/api/contract_test.go](../../retrocast/internal/api/contract_test.go) |
| SV-NFR-005 | 1 | `TestPilotRedisDownRouter//health` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-NFR-005 | 2 | `TestPilotRedisDownRouter//health` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-NFR-001 | 1 | `TestRegistrationAtomicityLive/pilot_stored_password` | [retrocast/internal/api/atomicity_live_test.go](../../retrocast/internal/api/atomicity_live_test.go) |
| SV-NFR-001 | 2 | `TestRegistrationAtomicityLive/pilot_stored_password` | [retrocast/internal/api/atomicity_live_test.go](../../retrocast/internal/api/atomicity_live_test.go) |
| SV-NFR-010 | 1 | `TestPilotRedisDownRouter` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-NFR-010 | 2 | `TestPilotRedisDownRouter` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-NFR-010 | 3 | `TestPilotRedisDownRouter` | [retrocast/internal/api/pilot_acceptance_test.go](../../retrocast/internal/api/pilot_acceptance_test.go) |
| SV-NFR-008 | 2 | `TestConstructorDomainAndOfflinePresign` | [retrocast/internal/storage/minio_test.go](../../retrocast/internal/storage/minio_test.go) |
| WEB-FR-001 | 1 | `Pilot server address persists and submit opens login` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-001 | 2 | `Pilot server address persists and submit opens login` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-001 | 3 | `Pilot stored server skips the address page on return` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-003 | 1 | `Pilot expired channel refreshes once and replaces stored token` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-003 | 2 | `Pilot expired channel refreshes once and replaces stored token` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-004 | 2 | `READY after RESUME signals resync and retains the replacement session` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-006 | 1 | `Pilot gateway channel, membership and idle updates drive real stores` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-006 | 2 | `Pilot gateway channel, membership and idle updates drive real stores; Pilot channel deletion accepts 204 and closes without an error` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-008 | 1 | `Pilot composer sends hi, clears and throttles twenty seconds of input` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-008 | 2 | `Pilot composer sends hi, clears and throttles twenty seconds of input` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-008 | 3 | `Pilot composer sends hi, clears and throttles twenty seconds of input` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-009 | 3 | `Pilot typing expires at eight seconds and renewed signals extend it` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-010 | 1 | `Pilot gateway channel, membership and idle updates drive real stores` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-011 | 1 | `Pilot markdown displays hostile markup literally without active attributes` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-011 | 2 | `Pilot markdown displays hostile markup literally without active attributes` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-014 | 1 | `Pilot leave removes the selected non-owner guild` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-020 | 2 | `Pilot registration explains invite-required and closed errors` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-020 | 3 | `Pilot registration explains invite-required and closed errors` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |
| WEB-FR-021 | 2 | `Pilot gateway channel, membership and idle updates drive real stores` | [retrocast-web/contract.test.mjs](../../retrocast-web/contract.test.mjs) |

Class a coverage: 82/82 clauses. No class b deployed checks are claimed here.

Additional coverage: SV-IR-001 uses `TestContractDocumentedSuccess`, which enumerates all 60 OpenAPI operations against seeded routed services (0 skipped; object storage is mocked). The same test checks exact data envelopes and empty 204 bodies. WEB-FR-009 clause 1 and member-list names use `Pilot member names resolve nickname then display name then username`; cache hydration uses `Pilot member responses and events hydrate the shared users cache`. The ALREADY_MEMBER store-hit branch uses `Pilot already-member store hit selects the previewed guild`. SV-NFR-004 uses `TestShutdownServerDeadline`; deployed SIGTERM remains with run 3.

Public member identity: `TestMemberPublicIdentityBatch` asserts two users and an unknown user with exactly one identity query, and restricts serialized fields to the public allowlist. `TestRegistrationAtomicityLive/pilot_defaults_member_identity_and_page` checks list/get/update and update events; `TestModerationAtomicityLive` checks add events. READY has guild ids, not member objects.
