import { create } from "zustand";
import { api } from "@/lib/api";
import type { Channel } from "@/types";

interface ChannelsState {
  channelsByGuild: Map<string, Channel[]>;
  selectedChannelId: string | null;
  setChannels: (guildId: string, channels: Channel[]) => void;
  addChannel: (channel: Channel) => void;
  updateChannel: (channel: Channel) => void;
  removeChannel: (channelId: string, guildId: string) => void;
  selectChannel: (channelId: string | null) => void;
  fetchChannels: (guildId: string) => Promise<void>;
}

export const useChannelsStore = create<ChannelsState>()((set) => ({
  channelsByGuild: new Map(),
  selectedChannelId: null,

  setChannels: (guildId, channels) => {
    set((state) => {
      const channelsByGuild = new Map(state.channelsByGuild);
      channelsByGuild.set(
        guildId,
        [...channels].sort((a, b) => a.position - b.position),
      );
      return { channelsByGuild };
    });
  },

  addChannel: (channel) => {
    set((state) => {
      const channelsByGuild = new Map(state.channelsByGuild);
      const existing = channelsByGuild.get(channel.guild_id) || [];
      channelsByGuild.set(
        channel.guild_id,
        [...existing, channel].sort((a, b) => a.position - b.position),
      );
      return { channelsByGuild };
    });
  },

  updateChannel: (channel) => {
    set((state) => {
      const channelsByGuild = new Map(state.channelsByGuild);
      const existing = channelsByGuild.get(channel.guild_id) || [];
      channelsByGuild.set(
        channel.guild_id,
        existing
          .map((c) => (c.id === channel.id ? channel : c))
          .sort((a, b) => a.position - b.position),
      );
      return { channelsByGuild };
    });
  },

  removeChannel: (channelId, guildId) => {
    set((state) => {
      const channelsByGuild = new Map(state.channelsByGuild);
      const existing = channelsByGuild.get(guildId) || [];
      channelsByGuild.set(
        guildId,
        existing.filter((c) => c.id !== channelId),
      );
      const selectedChannelId =
        state.selectedChannelId === channelId ? null : state.selectedChannelId;
      return { channelsByGuild, selectedChannelId };
    });
  },

  selectChannel: (channelId) => {
    set({ selectedChannelId: channelId });
  },

  fetchChannels: async (guildId) => {
    const channels = await api.get<Channel[]>(
      `/api/v1/guilds/${guildId}/channels`,
    );
    set((state) => {
      const channelsByGuild = new Map(state.channelsByGuild);
      channelsByGuild.set(
        guildId,
        [...channels].sort((a, b) => a.position - b.position),
      );
      return { channelsByGuild };
    });
  },
}));
