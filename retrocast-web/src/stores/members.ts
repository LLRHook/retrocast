import { create } from "zustand";
import { api } from "@/lib/api";
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
        guildMembers.set(member.user_id, member);
      }
      membersByGuild.set(guildId, guildMembers);
      return { membersByGuild };
    });
  },
}));
