import { useEffect } from "react";
import { api } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import { useGuildsStore } from "@/stores/guilds";
import { useMembersStore } from "@/stores/members";
import { useRolesStore } from "@/stores/roles";
import type { Member } from "@/types";

export function useModeration(guildId: string) {
 const userId = useAuthStore(s => s.user?.id);
 const ownerId = useGuildsStore(s => s.guilds.get(guildId)?.owner_id);
 const member = useMembersStore(s => userId ? s.membersByGuild.get(guildId)?.get(userId) : undefined);
 const roles = useRolesStore(s => s.rolesByGuild.get(guildId));
 useEffect(() => {
  let active = true;
  if (userId) api.get<Member>(`/api/v1/guilds/${guildId}/members/${userId}`).then(m => {
   if (active) useMembersStore.getState().setMember(m);
  }).catch(() => {});
  void useRolesStore.getState().fetchRoles(guildId).catch(() => {});
  return () => { active = false; };
 }, [guildId, userId]);
 const bits = (roles ?? []).filter(r => r.is_default || member?.roles.includes(r.id))
  .reduce((value, role) => value | BigInt(role.permissions), 0n);
 const all = !!userId && (userId === ownerId || (bits & (1n << 31n)) !== 0n);
 return { kick: all || (bits & (1n << 5n)) !== 0n, ban: all || (bits & (1n << 6n)) !== 0n, userId };
}
