import { create } from "zustand";
import { api } from "@/lib/api";
import type { User } from "@/types";

interface AuthResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

interface RefreshResponse {
  access_token: string;
}

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  serverUrl: string | null;
  isAuthenticated: boolean;
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  setServerUrl: (url: string) => void;
  refreshAccessToken: () => Promise<string>;
}

export const useAuthStore = create<AuthState>()((set, get) => {
  // Wire up the API client token provider
  api.setTokenProvider(() => get().accessToken);
  api.setTokenRefresher(() => get().refreshAccessToken());

  return {
    user: null,
    accessToken: null,
    refreshToken: localStorage.getItem("refreshToken"),
    serverUrl: localStorage.getItem("serverUrl"),
    isAuthenticated: false,

    login: async (username, password) => {
      const data = await api.post<AuthResponse>("/api/v1/auth/login", {
        username,
        password,
      });
      localStorage.setItem("refreshToken", data.refresh_token);
      set({
        user: data.user,
        accessToken: data.access_token,
        refreshToken: data.refresh_token,
        isAuthenticated: true,
      });
    },

    register: async (username, password) => {
      const data = await api.post<AuthResponse>("/api/v1/auth/register", {
        username,
        password,
      });
      localStorage.setItem("refreshToken", data.refresh_token);
      set({
        user: data.user,
        accessToken: data.access_token,
        refreshToken: data.refresh_token,
        isAuthenticated: true,
      });
    },

    logout: async () => {
      localStorage.removeItem("refreshToken");
      set({
        user: null,
        accessToken: null,
        refreshToken: null,
        isAuthenticated: false,
      });
    },

    setServerUrl: (url) => {
      localStorage.setItem("serverUrl", url);
      set({ serverUrl: url });
    },

    refreshAccessToken: async () => {
      const { refreshToken } = get();
      if (!refreshToken) {
        throw new Error("No refresh token");
      }
      const data = await api.post<RefreshResponse>("/api/v1/auth/refresh", {
        refresh_token: refreshToken,
      });
      set({ accessToken: data.access_token });
      return data.access_token;
    },
  };
});
