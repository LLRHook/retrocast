import { useEffect, useState, useSyncExternalStore } from "react";
import { Navigate, Outlet } from "react-router";
import { useAuthStore } from "@/stores/auth";
import { useGuildsStore } from "@/stores/guilds";
import { useChannelsStore } from "@/stores/channels";
import { useDMsStore } from "@/stores/dms";
import { gateway } from "@/lib/gateway";
import { initGatewayDispatcher } from "@/lib/gateway-dispatcher";
import ServerList from "@/components/ServerList";
import ChannelSidebar from "@/components/ChannelSidebar";

function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (cb) => {
      const mql = window.matchMedia(query);
      mql.addEventListener("change", cb);
      return () => mql.removeEventListener("change", cb);
    },
    () => window.matchMedia(query).matches,
    () => false,
  );
}

export default function AppLayout() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const accessToken = useAuthStore((s) => s.accessToken);
  const serverUrl = useAuthStore((s) => s.serverUrl);
  const isMobile = !useMediaQuery("(min-width: 768px)");
  const [sidebarOpen, setSidebarOpen] = useState(false);

  useEffect(() => {
    if (!isAuthenticated || !accessToken || !serverUrl) return;

    initGatewayDispatcher();

    // Connect gateway
    const wsUrl = serverUrl.replace(/^http/, "ws") + "/gateway";
    gateway.connect(wsUrl, accessToken);

    // Fetch initial data
    useGuildsStore
      .getState()
      .fetchGuilds()
      .then(() => {
        const { guilds, selectedGuildId, selectGuild } =
          useGuildsStore.getState();
        // Auto-select first guild if none selected
        if (!selectedGuildId && guilds.size > 0) {
          const firstGuild = guilds.values().next().value;
          if (firstGuild) {
            selectGuild(firstGuild.id);
            // Fetch channels for the first guild
            useChannelsStore
              .getState()
              .fetchChannels(firstGuild.id)
              .then(() => {
                const channels = useChannelsStore
                  .getState()
                  .channelsByGuild.get(firstGuild.id);
                const firstText = channels?.find((c) => c.type === 0);
                if (firstText) {
                  useChannelsStore.getState().selectChannel(firstText.id);
                }
              })
              .catch(() => {});
          }
        }
      })
      .catch(() => {});

    useDMsStore.getState().fetchDMs().catch(() => {});

    return () => {
      gateway.disconnect();
    };
  }, [isAuthenticated, accessToken, serverUrl]);

  // Close sidebar when a channel/DM is selected (mobile)
  useEffect(() => {
    if (!isMobile) return;
    let prevChannel = useChannelsStore.getState().selectedChannelId;
    let prevDM = useDMsStore.getState().selectedDMId;
    const unsubChannel = useChannelsStore.subscribe((s) => {
      if (s.selectedChannelId !== prevChannel) {
        prevChannel = s.selectedChannelId;
        setSidebarOpen(false);
      }
    });
    const unsubDM = useDMsStore.subscribe((s) => {
      if (s.selectedDMId !== prevDM) {
        prevDM = s.selectedDMId;
        setSidebarOpen(false);
      }
    });
    return () => {
      unsubChannel();
      unsubDM();
    };
  }, [isMobile]);

  if (!isAuthenticated) {
    return <Navigate to="/server" replace />;
  }

  return (
    <div className="flex h-screen bg-bg-primary text-text-primary">
      {/* Mobile hamburger button */}
      <button
        onClick={() => setSidebarOpen((v) => !v)}
        className="fixed left-2 top-2 z-40 flex h-8 w-8 items-center justify-center rounded bg-bg-secondary text-text-muted md:hidden"
        aria-label="Toggle sidebar"
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
          <path d="M3 18h18v-2H3v2Zm0-5h18v-2H3v2Zm0-7v2h18V6H3Z" />
        </svg>
      </button>

      {/* Sidebar overlay for mobile */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-30 bg-black/50 md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebars: always visible on md+, toggled on mobile */}
      <div
        className={`fixed inset-y-0 left-0 z-30 flex transition-transform md:relative md:translate-x-0 ${
          sidebarOpen ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <ServerList />
        <ChannelSidebar />
      </div>

      <main className="flex flex-1 flex-col overflow-hidden">
        <Outlet />
      </main>
    </div>
  );
}
