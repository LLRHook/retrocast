import { gateway } from "@/lib/gateway";
import { useGuildsStore } from "@/stores/guilds";
import { useChannelsStore } from "@/stores/channels";
import { useMembersStore } from "@/stores/members";
import { useRolesStore } from "@/stores/roles";
import { useMessagesStore } from "@/stores/messages";
import { usePresenceStore } from "@/stores/presence";
import { useTypingStore } from "@/stores/typing";
import { useUsersStore } from "@/stores/users";
import type { Guild, Channel, Member, Message, Role } from "@/types";

interface ReadyPayload {
  session_id: string;
  user_id: string;
  guilds: string[];
}

interface MessageDeletePayload {
  id: string;
  channel_id: string;
}

interface ChannelDeletePayload {
  id: string;
  guild_id: string;
}

interface GuildDeletePayload {
  id: string;
}

interface MemberRemovePayload {
  guild_id: string;
  user_id: string;
}

interface RoleEventPayload {
  guild_id: string;
  role: Role;
}

interface RoleDeletePayload {
  guild_id: string;
  role_id: string;
}

interface TypingPayload {
  channel_id: string;
  guild_id: string;
  user_id: string;
}

interface PresencePayload {
  user_id: string;
  status: "online" | "idle" | "dnd" | "invisible" | "offline";
}

let initialized = false;

export function initGatewayDispatcher() {
  if (initialized) return;
  initialized = true;

  gateway.on("READY", (_data) => {
    void (_data as ReadyPayload);
    // READY provides guild IDs — fetch full guild data
    useGuildsStore.getState().fetchGuilds().catch(() => {});
  });

  // Guild events
  gateway.on("GUILD_CREATE", (data) => {
    const guild = data as Guild;
    useGuildsStore.getState().setGuild(guild);
  });

  gateway.on("GUILD_UPDATE", (data) => {
    const guild = data as Guild;
    useGuildsStore.getState().setGuild(guild);
  });

  gateway.on("GUILD_DELETE", (data) => {
    const payload = data as GuildDeletePayload;
    useGuildsStore.getState().removeGuild(payload.id);
  });

  // Channel events
  gateway.on("CHANNEL_CREATE", (data) => {
    const channel = data as Channel;
    useChannelsStore.getState().addChannel(channel);
  });

  gateway.on("CHANNEL_UPDATE", (data) => {
    const channel = data as Channel;
    useChannelsStore.getState().updateChannel(channel);
  });

  gateway.on("CHANNEL_DELETE", (data) => {
    const payload = data as ChannelDeletePayload;
    useChannelsStore.getState().removeChannel(payload.id, payload.guild_id);
  });

  // Message events
  gateway.on("MESSAGE_CREATE", (data) => {
    const message = data as Message;
    useMessagesStore.getState().addMessage(message);
    useUsersStore.getState().cacheFromMessage(message);
    // Clear typing for the message author
    useTypingStore.getState().clearTyping(message.channel_id, message.author_id);
  });

  gateway.on("MESSAGE_UPDATE", (data) => {
    const message = data as Message;
    useMessagesStore.getState().updateMessage(message);
    useUsersStore.getState().cacheFromMessage(message);
  });

  gateway.on("MESSAGE_DELETE", (data) => {
    const payload = data as MessageDeletePayload;
    useMessagesStore.getState().removeMessage(payload.channel_id, payload.id);
  });

  // Member events
  gateway.on("GUILD_MEMBER_ADD", (data) => {
    const member = data as Member;
    useMembersStore.getState().setMember(member);
  });

  gateway.on("GUILD_MEMBER_UPDATE", (data) => {
    const member = data as Member;
    useMembersStore.getState().setMember(member);
  });

  gateway.on("GUILD_MEMBER_REMOVE", (data) => {
    const payload = data as MemberRemovePayload;
    useMembersStore.getState().removeMember(payload.guild_id, payload.user_id);
  });

  // Role events
  gateway.on("GUILD_ROLE_CREATE", (data) => {
    const payload = data as RoleEventPayload;
    useRolesStore.getState().setRole(payload.role);
  });

  gateway.on("GUILD_ROLE_UPDATE", (data) => {
    const payload = data as RoleEventPayload;
    useRolesStore.getState().setRole(payload.role);
  });

  gateway.on("GUILD_ROLE_DELETE", (data) => {
    const payload = data as RoleDeletePayload;
    useRolesStore.getState().removeRole(payload.guild_id, payload.role_id);
  });

  // Typing
  gateway.on("TYPING_START", (data) => {
    const payload = data as TypingPayload;
    useTypingStore.getState().setTyping(payload.channel_id, payload.user_id);
  });

  // Presence
  gateway.on("PRESENCE_UPDATE", (data) => {
    const payload = data as PresencePayload;
    usePresenceStore.getState().setPresence(payload.user_id, payload.status);
  });

  // Ban events — no client-side store needed yet, but could show toasts
  gateway.on("GUILD_BAN_ADD", () => {});
  gateway.on("GUILD_BAN_REMOVE", () => {});

  // Reaction events — placeholder for future store
  gateway.on("MESSAGE_REACTION_ADD", () => {});
  gateway.on("MESSAGE_REACTION_REMOVE", () => {});
}
