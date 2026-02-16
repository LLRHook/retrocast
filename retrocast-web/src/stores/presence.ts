import { create } from "zustand";

type PresenceStatus = "online" | "idle" | "dnd" | "invisible" | "offline";

interface PresenceState {
  presences: Map<string, PresenceStatus>;
  setPresence: (userId: string, status: PresenceStatus) => void;
  clearPresences: () => void;
}

export const usePresenceStore = create<PresenceState>()((set) => ({
  presences: new Map(),

  setPresence: (userId, status) => {
    set((state) => {
      const presences = new Map(state.presences);
      presences.set(userId, status);
      return { presences };
    });
  },

  clearPresences: () => {
    set({ presences: new Map() });
  },
}));
