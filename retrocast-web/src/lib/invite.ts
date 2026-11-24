import { api } from "@/lib/api";
import { useGuildsStore } from "@/stores/guilds";
import { useChannelsStore } from "@/stores/channels";
import type { Guild } from "@/types";

export async function selectInviteGuild(guild: Guild) {
  useGuildsStore.getState().setGuild(guild);
  useGuildsStore.getState().selectGuild(guild.id);
  useChannelsStore.getState().selectChannel(null);
  await useChannelsStore.getState().fetchChannels(guild.id);
  const channel = useChannelsStore.getState().channelsByGuild.get(guild.id)?.find(c => c.type === 0);
  if (channel) useChannelsStore.getState().selectChannel(channel.id);
}

// React StrictMode can replay effects; a shared flight avoids consuming two uses.
const flights = new Map<string, Promise<void>>();
export function acceptInvite(code: string) {
  const active = flights.get(code);
  if (active) return active;
  const flight = api.post<Guild>(`/api/v1/invites/${encodeURIComponent(code)}`).then(selectInviteGuild)
    .finally(() => flights.delete(code));
  flights.set(code, flight);
  return flight;
}
