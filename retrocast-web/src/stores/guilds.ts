import { create } from "zustand";
import { api } from "@/lib/api";
import type { Guild } from "@/types";

interface GuildsState {
  guilds: Map<string, Guild>;
  selectedGuildId: string | null;
  setGuild: (guild: Guild) => void;
  removeGuild: (guildId: string) => void;
  selectGuild: (guildId: string | null) => void;
  fetchGuilds: () => Promise<void>;
  createGuild: (name: string) => Promise<Guild>;
}

export const useGuildsStore = create<GuildsState>()((set, get) => ({
  guilds: new Map(),
  selectedGuildId: null,

  setGuild: (guild) => {
    set((state) => {
      const guilds = new Map(state.guilds);
      guilds.set(guild.id, guild);
      return { guilds };
    });
  },

  removeGuild: (guildId) => {
    set((state) => {
      const guilds = new Map(state.guilds);
      guilds.delete(guildId);
      const selectedGuildId =
        state.selectedGuildId === guildId ? null : state.selectedGuildId;
      return { guilds, selectedGuildId };
    });
  },

  selectGuild: (guildId) => {
    set({ selectedGuildId: guildId });
  },

  fetchGuilds: async () => {
    const guilds = await api.get<Guild[]>("/api/v1/users/@me/guilds");
    const map = new Map<string, Guild>();
    for (const guild of guilds) {
      map.set(guild.id, guild);
    }
    set({ guilds: map });
  },

  createGuild: async (name) => {
    const guild = await api.post<Guild>("/api/v1/guilds", { name });
    get().setGuild(guild);
    return guild;
  },
}));
