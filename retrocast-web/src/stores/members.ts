import { create } from "zustand";
import { api } from "@/lib/api";
import { useUsersStore } from "@/stores/users";
import type { Member } from "@/types";

interface MembersState {
  membersByGuild: Map<string, Map<string, Member>>;
  setMember: (member: Member) => void;
  removeMember: (guildId: string, userId: string) => void;
  fetchMembers: (guildId: string) => Promise<void>;
}

export const useMembersStore = create<MembersState>()((set) => ({
  membersByGuild: new Map(),

  setMember: (member) => {
    if (member.user) useUsersStore.getState().cacheUser(member.user.id, {
      username: member.user.username, display_name: member.user.display_name,
      avatar_hash: member.user.avatar_hash ?? null,
    });
    set((state) => {
      const membersByGuild = new Map(state.membersByGuild);
      const guildMembers = new Map(
        membersByGuild.get(member.guild_id) || new Map(),
      );
      guildMembers.set(member.user_id, member);
      membersByGuild.set(member.guild_id, guildMembers);
      return { membersByGuild };
    });
  },

  removeMember: (guildId, userId) => {
    set((state) => {
      const membersByGuild = new Map(state.membersByGuild);
      const guildMembers = new Map(
        membersByGuild.get(guildId) || new Map(),
      );
      guildMembers.delete(userId);
      membersByGuild.set(guildId, guildMembers);
      return { membersByGuild };
    });
  },

  fetchMembers: async (guildId) => {
    const members = await api.get<Member[]>(
      `/api/v1/guilds/${guildId}/members`,
    );
    set((state) => {
      const membersByGuild = new Map(state.membersByGuild);
      const guildMembers = new Map<string, Member>();
      for (const member of members) {
        if (member.user) useUsersStore.getState().cacheUser(member.user.id, {
          username: member.user.username, display_name: member.user.display_name,
          avatar_hash: member.user.avatar_hash ?? null,
        });
        guildMembers.set(member.user_id, member);
      }
      membersByGuild.set(guildId, guildMembers);
      return { membersByGuild };
    });
  },
}));
