import { api, ApiClientError } from "@/lib/api";
import { useGuildsStore } from "@/stores/guilds";
import { useChannelsStore } from "@/stores/channels";
import type { Guild } from "@/types";

// All server entry points share selection and reject late channel responses.
export async function selectGuildChannels(guildId: string | null) {
  useGuildsStore.getState().selectGuild(guildId);
  useChannelsStore.getState().selectChannel(null);
  if (!guildId) return;
  if (!useChannelsStore.getState().channelsByGuild.has(guildId)) {
    await useChannelsStore.getState().fetchChannels(guildId);
  }
  if (useGuildsStore.getState().selectedGuildId !== guildId) return;
  const channel = useChannelsStore.getState().channelsByGuild.get(guildId)?.find(c => c.type === 0);
  useChannelsStore.getState().selectChannel(channel?.id ?? null);
}

export async function selectInviteGuild(guild: Guild) {
  useGuildsStore.getState().setGuild(guild);
  await selectGuildChannels(guild.id);
  return guild;
}

// React StrictMode can replay effects; a shared flight avoids consuming two uses.
const flights = new Map<string, Promise<Guild>>();
export function acceptInvite(code: string) {
  const active = flights.get(code);
  if (active) return active;
  const flight = api.post<Guild>(`/api/v1/invites/${encodeURIComponent(code)}`).catch(async (err: unknown) => {
      if (!(err instanceof ApiClientError) || err.code !== "ALREADY_MEMBER") throw err;
      // The POST may already have committed in another tab or before a reload.
      // Preview identifies the exact destination without consuming another use.
      const preview = await api.get<{guild_id: string}>(`/api/v1/invites/${encodeURIComponent(code)}`);
      return useGuildsStore.getState().guilds.get(preview.guild_id)
        ?? await api.get<Guild>(`/api/v1/guilds/${encodeURIComponent(preview.guild_id)}`);
    }).then(selectInviteGuild)
    .finally(() => flights.delete(code));
  flights.set(code, flight);
  return flight;
}
