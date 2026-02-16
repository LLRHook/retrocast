import { create } from "zustand";
import { api } from "@/lib/api";
import type { Role } from "@/types";

interface RolesState {
  rolesByGuild: Map<string, Role[]>;
  setRole: (role: Role) => void;
  removeRole: (guildId: string, roleId: string) => void;
  setRoles: (guildId: string, roles: Role[]) => void;
  fetchRoles: (guildId: string) => Promise<void>;
}

export const useRolesStore = create<RolesState>()((set) => ({
  rolesByGuild: new Map(),

  setRole: (role) => {
    set((state) => {
      const rolesByGuild = new Map(state.rolesByGuild);
      const existing = rolesByGuild.get(role.guild_id) || [];
      const idx = existing.findIndex((r) => r.id === role.id);
      const updated =
        idx >= 0
          ? existing.map((r) => (r.id === role.id ? role : r))
          : [...existing, role];
      rolesByGuild.set(
        role.guild_id,
        updated.sort((a, b) => a.position - b.position),
      );
      return { rolesByGuild };
    });
  },

  removeRole: (guildId, roleId) => {
    set((state) => {
      const rolesByGuild = new Map(state.rolesByGuild);
      const existing = rolesByGuild.get(guildId) || [];
      rolesByGuild.set(
        guildId,
        existing.filter((r) => r.id !== roleId),
      );
      return { rolesByGuild };
    });
  },

  setRoles: (guildId, roles) => {
    set((state) => {
      const rolesByGuild = new Map(state.rolesByGuild);
      rolesByGuild.set(
        guildId,
        [...roles].sort((a, b) => a.position - b.position),
      );
      return { rolesByGuild };
    });
  },

  fetchRoles: async (guildId) => {
    const roles = await api.get<Role[]>(`/api/v1/guilds/${guildId}/roles`);
    set((state) => {
      const rolesByGuild = new Map(state.rolesByGuild);
      rolesByGuild.set(
        guildId,
        [...roles].sort((a, b) => a.position - b.position),
      );
      return { rolesByGuild };
    });
  },
}));
