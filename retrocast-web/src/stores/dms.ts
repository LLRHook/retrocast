import { create } from "zustand";
import { api } from "@/lib/api";
import { useUsersStore } from "@/stores/users";
import type { DMChannel } from "@/types";

interface DMsState {
  dms: Map<string, DMChannel>;
  showDMList: boolean;
  selectedDMId: string | null;
  fetchDMs: () => Promise<void>;
  addDM: (dm: DMChannel) => void;
  selectDM: (dmId: string | null) => void;
  toggleDMList: () => void;
  setShowDMList: (show: boolean) => void;
}

export const useDMsStore = create<DMsState>()((set) => ({
  dms: new Map(),
  showDMList: false,
  selectedDMId: null,

  fetchDMs: async () => {
    const dms = await api.get<DMChannel[]>("/api/v1/users/@me/channels");
    const map = new Map<string, DMChannel>();
    const usersStore = useUsersStore.getState();
    for (const dm of dms) {
      map.set(dm.id, dm);
      // Cache DM recipients in user store
      for (const recipient of dm.recipients) {
        usersStore.cacheFromUser(recipient);
      }
    }
    set({ dms: map });
  },

  addDM: (dm) => {
    set((state) => {
      const dms = new Map(state.dms);
      dms.set(dm.id, dm);
      return { dms };
    });
  },

  selectDM: (dmId) => {
    set({ selectedDMId: dmId });
  },

  toggleDMList: () => {
    set((state) => ({ showDMList: !state.showDMList }));
  },

  setShowDMList: (show) => {
    set({ showDMList: show });
  },
}));
