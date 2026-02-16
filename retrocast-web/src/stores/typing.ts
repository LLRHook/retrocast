import { create } from "zustand";

interface TypingEntry {
  userId: string;
  timestamp: number;
}

interface TypingState {
  typingByChannel: Map<string, Map<string, TypingEntry>>;
  setTyping: (channelId: string, userId: string) => void;
  clearTyping: (channelId: string, userId: string) => void;
}

const TYPING_TIMEOUT = 8000;

export const useTypingStore = create<TypingState>()((set) => ({
  typingByChannel: new Map(),

  setTyping: (channelId, userId) => {
    set((state) => {
      const typingByChannel = new Map(state.typingByChannel);
      const channelTyping = new Map(typingByChannel.get(channelId) || new Map());
      channelTyping.set(userId, { userId, timestamp: Date.now() });
      typingByChannel.set(channelId, channelTyping);
      return { typingByChannel };
    });

    // Auto-clear after timeout
    setTimeout(() => {
      set((state) => {
        const typingByChannel = new Map(state.typingByChannel);
        const channelTyping = new Map(
          typingByChannel.get(channelId) || new Map(),
        );
        const entry = channelTyping.get(userId);
        if (entry && Date.now() - entry.timestamp >= TYPING_TIMEOUT) {
          channelTyping.delete(userId);
          typingByChannel.set(channelId, channelTyping);
        }
        return { typingByChannel };
      });
    }, TYPING_TIMEOUT);
  },

  clearTyping: (channelId, userId) => {
    set((state) => {
      const typingByChannel = new Map(state.typingByChannel);
      const channelTyping = new Map(
        typingByChannel.get(channelId) || new Map(),
      );
      channelTyping.delete(userId);
      typingByChannel.set(channelId, channelTyping);
      return { typingByChannel };
    });
  },
}));
