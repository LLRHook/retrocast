import { create } from "zustand";
import { api, ApiClientError } from "@/lib/api";
import type { User } from "@/types";

interface AuthResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

interface RefreshResponse {
  access_token: string;
  refresh_token: string;
}

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  serverUrl: string | null;
  isAuthenticated: boolean;
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, password: string, inviteCode?: string) => Promise<void>;
  restore: () => Promise<void>;
  restoreState: "pending" | "ready" | "error";
  changePassword: (current: string, next: string) => Promise<void>;
  logout: () => Promise<void>;
  setServerUrl: (url: string) => void;
  refreshAccessToken: () => Promise<string>;
}

// Web Locks serialize rotations across same-origin tabs; each lock reads the latest
// token from storage. The store promise also coalesces API, restore and logout calls.
async function sessionLock<T>(action: () => Promise<T>): Promise<T> {
  if (globalThis.navigator?.locks) return navigator.locks.request("retrocast-session", action);
  return action();
}
let refreshFlight: Promise<string> | null = null;
let restoreFlight: Promise<void> | null = null;
// Keep the attempted token private, associated with the exact failed request.
const rejectedTokens = new WeakMap<Error, string>();
export const useAuthStore = create<AuthState>()((set, get) => {
  // Wire up the API client token provider
  api.setTokenProvider(() => get().accessToken);
  api.setTokenRefresher(() => get().refreshAccessToken());

  const rotate = async () => {
    const refreshToken = localStorage.getItem("refreshToken");
    if (!refreshToken) throw new Error("No refresh token");
    try {
      const data = await api.post<RefreshResponse>("/api/v1/auth/refresh", {refresh_token: refreshToken});
      // A newer session (including a password change) wins over a late response.
      if (localStorage.getItem("refreshToken") !== refreshToken) {
        const latest = localStorage.getItem("refreshToken");
        if (!latest) throw new Error("Session changed");
        return get().accessToken ?? data.access_token;
      }
      localStorage.setItem("refreshToken", data.refresh_token);
      set({accessToken: data.access_token, refreshToken: data.refresh_token});
      return data.access_token;
    } catch (err) {
      if (err instanceof ApiClientError && err.status === 401) rejectedTokens.set(err, refreshToken);
      throw err;
    }
  };

  return {
    restoreState: "pending",
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

    register: async (username, password, inviteCode) => {
      const data = await api.post<AuthResponse>("/api/v1/auth/register", {
        invite_code: inviteCode || undefined,
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
      // Drain an existing local refresh before taking the shared tab lock.
      if (refreshFlight) { try { await refreshFlight; } catch { /* best effort */ } }
      await sessionLock(async () => {
        try {
          const refreshToken = localStorage.getItem("refreshToken");
          if (refreshToken) {
            try { await api.post("/api/v1/auth/logout", {refresh_token: refreshToken}); }
            catch (err) {
              if (!(err instanceof ApiClientError) || err.status !== 401) throw err;
              // Already under the same lock: rotate directly to avoid nesting it.
              await rotate();
              await api.post("/api/v1/auth/logout", {refresh_token: get().refreshToken});
            }
          }
        } catch { console.warn("Could not revoke the server session while signing out"); }
        finally {
          localStorage.removeItem("refreshToken");
          set({user: null, accessToken: null, refreshToken: null, isAuthenticated: false});
        }
      });
    },

    setServerUrl: (url) => {
      localStorage.setItem("serverUrl", url);
      set({ serverUrl: url });
    },

    changePassword: async (current, next) => {
      const data = await api.put<AuthResponse>("/api/v1/users/@me/password", {
        current_password: current, new_password: next,
      });
      await sessionLock(async () => {
        localStorage.setItem("refreshToken", data.refresh_token);
        set({user: data.user, accessToken: data.access_token, refreshToken: data.refresh_token});
      });
    },

    restore: () => {
      if (restoreFlight) return restoreFlight;
      restoreFlight = (async () => {
        if (!get().serverUrl || !get().refreshToken) { set({restoreState: "ready"}); return; }
        set({restoreState: "pending"});
        // One retry handles a rotation won by another tab without Web Locks.
        for (let attempt = 0; attempt < 2; attempt++) {
          try {
            await get().refreshAccessToken();
            const user = await api.get<User>("/api/v1/users/@me");
            set({user, isAuthenticated: true, restoreState: "ready"});
            return;
          } catch (err) {
            if (!(err instanceof ApiClientError) || err.status !== 401) {
              set({restoreState: "error"});
              return;
            }
            const rejected = rejectedTokens.get(err) ?? get().refreshToken;
            const replaced = await sessionLock(async () => {
              let stored = localStorage.getItem("refreshToken");
              if (!globalThis.navigator?.locks && (!stored || stored === rejected)) {
                await new Promise(resolve => setTimeout(resolve, 250));
                stored = localStorage.getItem("refreshToken");
              }
              if (stored && stored !== rejected) return true;
              localStorage.removeItem("refreshToken");
              set({user: null, accessToken: null, refreshToken: null, isAuthenticated: false, restoreState: "ready"});
              return false;
            });
            if (!replaced) return;
            // A second replacement is preserved for Retry instead of clearing it.
            if (attempt === 1) set({restoreState: "error"});
          }
        }
      })().finally(() => {restoreFlight = null;});
      return restoreFlight;
    },

    refreshAccessToken: () => {
      if (refreshFlight) return refreshFlight;
      refreshFlight = sessionLock(rotate).finally(() => {refreshFlight = null;});
      return refreshFlight;
    },
  };
});
