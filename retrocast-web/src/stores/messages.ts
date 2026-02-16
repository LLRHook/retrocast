import { create } from "zustand";
import { api } from "@/lib/api";
import { useUsersStore } from "@/stores/users";
import type { Message } from "@/types";

function cacheUsersFromMessages(messages: Message[]) {
  const store = useUsersStore.getState();
  for (const msg of messages) {
    store.cacheFromMessage(msg);
  }
}

const PAGE_SIZE = 50;

interface MessagesState {
  messagesByChannel: Map<string, Message[]>;
  hasMore: Map<string, boolean>;
  addMessage: (message: Message) => void;
  updateMessage: (message: Message) => void;
  removeMessage: (channelId: string, messageId: string) => void;
  setMessages: (channelId: string, messages: Message[]) => void;
  prependMessages: (channelId: string, messages: Message[]) => void;
  fetchMessages: (channelId: string, before?: string) => Promise<Message[]>;
}

export const useMessagesStore = create<MessagesState>()((set) => ({
  messagesByChannel: new Map(),
  hasMore: new Map(),

  addMessage: (message) => {
    set((state) => {
      const messagesByChannel = new Map(state.messagesByChannel);
      const existing = messagesByChannel.get(message.channel_id) || [];
      messagesByChannel.set(message.channel_id, [...existing, message]);
      return { messagesByChannel };
    });
  },

  updateMessage: (message) => {
    set((state) => {
      const messagesByChannel = new Map(state.messagesByChannel);
      const existing = messagesByChannel.get(message.channel_id) || [];
      messagesByChannel.set(
        message.channel_id,
        existing.map((m) => (m.id === message.id ? message : m)),
      );
      return { messagesByChannel };
    });
  },

  removeMessage: (channelId, messageId) => {
    set((state) => {
      const messagesByChannel = new Map(state.messagesByChannel);
      const existing = messagesByChannel.get(channelId) || [];
      messagesByChannel.set(
        channelId,
        existing.filter((m) => m.id !== messageId),
      );
      return { messagesByChannel };
    });
  },

  setMessages: (channelId, messages) => {
    cacheUsersFromMessages(messages);
    set((state) => {
      const messagesByChannel = new Map(state.messagesByChannel);
      messagesByChannel.set(channelId, messages);
      const hasMore = new Map(state.hasMore);
      hasMore.set(channelId, messages.length >= PAGE_SIZE);
      return { messagesByChannel, hasMore };
    });
  },

  prependMessages: (channelId, messages) => {
    cacheUsersFromMessages(messages);
    set((state) => {
      const messagesByChannel = new Map(state.messagesByChannel);
      const existing = messagesByChannel.get(channelId) || [];
      messagesByChannel.set(channelId, [...messages, ...existing]);
      const hasMore = new Map(state.hasMore);
      hasMore.set(channelId, messages.length >= PAGE_SIZE);
      return { messagesByChannel, hasMore };
    });
  },

  fetchMessages: async (channelId, before?) => {
    const params = before ? `?before=${before}` : "";
    const messages = await api.get<Message[]>(
      `/api/v1/channels/${channelId}/messages${params}`,
    );
    return messages;
  },
}));
