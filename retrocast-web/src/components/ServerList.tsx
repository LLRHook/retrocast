import { useState } from "react";
import { useGuildsStore } from "@/stores/guilds";
import { useChannelsStore } from "@/stores/channels";
import { useDMsStore } from "@/stores/dms";
import { api } from "@/lib/api";

function GuildIcon({
  guild,
  isSelected,
  onClick,
}: {
  guild: { id: string; name: string; icon_hash: string | null };
  isSelected: boolean;
  onClick: () => void;
}) {
  const initials = guild.name
    .split(" ")
    .map((w) => w[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();

  const serverUrl = localStorage.getItem("serverUrl") || "";

  return (
    <div className="group relative flex items-center justify-center">
      {/* Selection indicator */}
      <div
        className={`absolute left-0 w-1 rounded-r-sm bg-white transition-all ${
          isSelected ? "h-10" : "h-0 group-hover:h-5"
        }`}
      />
      <button
        onClick={onClick}
        className={`flex h-12 w-12 items-center justify-center transition-all ${
          isSelected
            ? "rounded-2xl bg-accent"
            : "rounded-3xl bg-bg-primary hover:rounded-2xl hover:bg-accent"
        }`}
        title={guild.name}
      >
        {guild.icon_hash ? (
          <img
            src={`${serverUrl}/api/v1/guilds/${guild.id}/icon`}
            alt={guild.name}
            className="h-12 w-12 rounded-[inherit] object-cover"
          />
        ) : (
          <span className="text-sm font-medium text-text-primary">
            {initials}
          </span>
        )}
      </button>
    </div>
  );
}

function CreateGuildModal({ onClose }: { onClose: () => void }) {
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const createGuild = useGuildsStore((s) => s.createGuild);
  const selectGuild = useGuildsStore((s) => s.selectGuild);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setLoading(true);
    setError("");
    try {
      const guild = await createGuild(name.trim());
      selectGuild(guild.id);
      onClose();
    } catch {
      setError("Failed to create server");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70"
      onClick={onClose}
    >
      <div
        className="w-full max-w-md rounded-lg bg-bg-primary p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-4 text-xl font-bold text-text-primary">
          Create a Server
        </h2>
        {error && (
          <div className="mb-3 rounded bg-red-500/10 p-2 text-sm text-red-400">
            {error}
          </div>
        )}
        <form onSubmit={handleCreate}>
          <label className="mb-2 block text-xs font-semibold uppercase text-text-secondary">
            Server Name
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="mb-4 w-full rounded bg-bg-input p-2.5 text-text-primary outline-none focus:ring-2 focus:ring-accent"
            autoFocus
          />
          <div className="flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="rounded px-4 py-2 text-sm text-text-secondary hover:text-text-primary"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !name.trim()}
              className="rounded bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent-hover disabled:opacity-50"
            >
              {loading ? "Creating..." : "Create"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function JoinGuildModal({ onClose }: { onClose: () => void }) {
  const [code, setCode] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const selectGuild = useGuildsStore((s) => s.selectGuild);

  async function handleJoin(e: React.FormEvent) {
    e.preventDefault();
    if (!code.trim()) return;
    setLoading(true);
    setError("");
    try {
      const result = await api.post<{ guild_id: string }>(
        `/api/v1/invites/${code.trim()}`,
      );
      // Refresh guild list
      await useGuildsStore.getState().fetchGuilds();
      selectGuild(result.guild_id);
      onClose();
    } catch {
      setError("Invalid or expired invite code");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70"
      onClick={onClose}
    >
      <div
        className="w-full max-w-md rounded-lg bg-bg-primary p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-4 text-xl font-bold text-text-primary">
          Join a Server
        </h2>
        {error && (
          <div className="mb-3 rounded bg-red-500/10 p-2 text-sm text-red-400">
            {error}
          </div>
        )}
        <form onSubmit={handleJoin}>
          <label className="mb-2 block text-xs font-semibold uppercase text-text-secondary">
            Invite Code
          </label>
          <input
            type="text"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder="abc123"
            className="mb-4 w-full rounded bg-bg-input p-2.5 text-text-primary outline-none focus:ring-2 focus:ring-accent"
            autoFocus
          />
          <div className="flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="rounded px-4 py-2 text-sm text-text-secondary hover:text-text-primary"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !code.trim()}
              className="rounded bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent-hover disabled:opacity-50"
            >
              {loading ? "Joining..." : "Join"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default function ServerList() {
  const guilds = useGuildsStore((s) => s.guilds);
  const selectedGuildId = useGuildsStore((s) => s.selectedGuildId);
  const selectGuild = useGuildsStore((s) => s.selectGuild);
  const fetchChannels = useChannelsStore((s) => s.fetchChannels);
  const selectChannel = useChannelsStore((s) => s.selectChannel);
  const channelsByGuild = useChannelsStore((s) => s.channelsByGuild);
  const showDMList = useDMsStore((s) => s.showDMList);
  const setShowDMList = useDMsStore((s) => s.setShowDMList);
  const [showCreate, setShowCreate] = useState(false);
  const [showJoin, setShowJoin] = useState(false);

  function handleDMClick() {
    setShowDMList(true);
    selectGuild(null);
    selectChannel(null);
  }

  function handleSelectGuild(guildId: string) {
    setShowDMList(false);
    selectGuild(guildId);
    // Fetch channels if not already loaded
    if (!channelsByGuild.has(guildId)) {
      fetchChannels(guildId).then(() => {
        // Auto-select first text channel
        const channels = useChannelsStore.getState().channelsByGuild.get(guildId);
        const firstText = channels?.find((c) => c.type === 0);
        if (firstText) {
          selectChannel(firstText.id);
        }
      });
    } else {
      // Auto-select first text channel if none selected
      const channels = channelsByGuild.get(guildId);
      const firstText = channels?.find((c) => c.type === 0);
      if (firstText) {
        selectChannel(firstText.id);
      }
    }
  }

  const guildList = Array.from(guilds.values());

  return (
    <div className="flex w-[72px] shrink-0 flex-col items-center gap-2 overflow-y-auto bg-bg-tertiary py-3">
      {/* DM button */}
      <div className="group relative flex items-center justify-center">
        <div
          className={`absolute left-0 w-1 rounded-r-sm bg-white transition-all ${
            showDMList ? "h-10" : "h-0 group-hover:h-5"
          }`}
        />
        <button
          onClick={handleDMClick}
          className={`flex h-12 w-12 items-center justify-center transition-all ${
            showDMList
              ? "rounded-2xl bg-accent"
              : "rounded-3xl bg-bg-primary hover:rounded-2xl hover:bg-accent"
          }`}
          title="Direct Messages"
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
            <path d="M19.73 4.87a18.2 18.2 0 0 0-4.6-1.44c-.2.36-.43.85-.59 1.23a16.84 16.84 0 0 0-5.07 0c-.16-.38-.4-.87-.6-1.23a18.17 18.17 0 0 0-4.6 1.44A19.25 19.25 0 0 0 .96 18.06a18.37 18.37 0 0 0 5.63 2.87c.46-.62.86-1.28 1.2-1.98a11.81 11.81 0 0 1-1.89-.92c.16-.12.31-.24.46-.36a12.93 12.93 0 0 0 11.27 0c.15.12.3.25.46.36-.6.36-1.23.67-1.89.92.35.7.75 1.36 1.2 1.98a18.3 18.3 0 0 0 5.63-2.87A19.2 19.2 0 0 0 19.73 4.87ZM8.3 15.12c-1.18 0-2.16-1.1-2.16-2.44 0-1.35.95-2.45 2.16-2.45 1.22 0 2.19 1.1 2.16 2.45 0 1.35-.95 2.44-2.16 2.44Zm7.4 0c-1.18 0-2.16-1.1-2.16-2.44 0-1.35.95-2.45 2.16-2.45 1.22 0 2.19 1.1 2.16 2.45 0 1.35-.94 2.44-2.16 2.44Z" />
          </svg>
        </button>
      </div>

      {/* Separator */}
      <div className="mx-auto h-0.5 w-8 rounded bg-border" />

      {guildList.map((guild) => (
        <GuildIcon
          key={guild.id}
          guild={guild}
          isSelected={selectedGuildId === guild.id}
          onClick={() => handleSelectGuild(guild.id)}
        />
      ))}

      {/* Separator */}
      <div className="mx-auto h-0.5 w-8 rounded bg-border" />

      {/* Create guild button */}
      <button
        onClick={() => setShowCreate(true)}
        className="flex h-12 w-12 items-center justify-center rounded-3xl bg-bg-primary text-green-500 transition-all hover:rounded-2xl hover:bg-green-500 hover:text-white"
        title="Create a Server"
      >
        <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
          <path d="M13 5a1 1 0 1 0-2 0v6H5a1 1 0 1 0 0 2h6v6a1 1 0 1 0 2 0v-6h6a1 1 0 1 0 0-2h-6V5Z" />
        </svg>
      </button>

      {/* Join guild button */}
      <button
        onClick={() => setShowJoin(true)}
        className="flex h-12 w-12 items-center justify-center rounded-3xl bg-bg-primary text-green-500 transition-all hover:rounded-2xl hover:bg-green-500 hover:text-white"
        title="Join a Server"
      >
        <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
          <path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2Z" />
        </svg>
      </button>

      {showCreate && <CreateGuildModal onClose={() => setShowCreate(false)} />}
      {showJoin && <JoinGuildModal onClose={() => setShowJoin(false)} />}
    </div>
  );
}
