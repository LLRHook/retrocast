import { create } from "zustand";
import type { User } from "@/types";

interface UserCache {
  username: string;
  display_name: string;
  avatar_hash: string | null;
}

interface UsersState {
  users: Map<string, UserCache>;
  cacheUser: (userId: string, data: UserCache) => void;
  cacheFromMessage: (msg: {
    author_id: string;
    author_username: string;
    author_display_name: string;
    author_avatar_hash: string | null;
  }) => void;
  cacheFromUser: (user: User) => void;
  getUser: (userId: string) => UserCache | undefined;
}

export const useUsersStore = create<UsersState>()((set, get) => ({
  users: new Map(),

  cacheUser: (userId, data) => {
    set((state) => {
      const users = new Map(state.users);
      users.set(userId, data);
      return { users };
    });
  },

  cacheFromMessage: (msg) => {
    set((state) => {
      const users = new Map(state.users);
      users.set(msg.author_id, {
        username: msg.author_username,
        display_name: msg.author_display_name,
        avatar_hash: msg.author_avatar_hash,
      });
      return { users };
    });
  },

  cacheFromUser: (user) => {
    set((state) => {
      const users = new Map(state.users);
      users.set(user.id, {
        username: user.username,
        display_name: user.display_name,
        avatar_hash: user.avatar_hash,
      });
      return { users };
    });
  },

  getUser: (userId) => {
    return get().users.get(userId);
  },
}));
