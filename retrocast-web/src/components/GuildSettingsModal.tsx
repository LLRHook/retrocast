import { useState } from "react";
import { api } from "@/lib/api";
import { useGuildsStore } from "@/stores/guilds";
import { useAuthStore } from "@/stores/auth";
import RoleEditor from "@/components/RoleEditor";
import type { Guild } from "@/types";

interface GuildSettingsModalProps {
  guild: Guild;
  onClose: () => void;
}

type Tab = "general" | "roles";

export default function GuildSettingsModal({
  guild,
  onClose,
}: GuildSettingsModalProps) {
  const [tab, setTab] = useState<Tab>("general");
  const [name, setName] = useState(guild.name);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const currentUser = useAuthStore((s) => s.user);
  const isOwner = currentUser?.id === guild.owner_id;
  const setGuild = useGuildsStore((s) => s.setGuild);
  const removeGuild = useGuildsStore((s) => s.removeGuild);
  const selectGuild = useGuildsStore((s) => s.selectGuild);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed || trimmed === guild.name) {
      onClose();
      return;
    }
    setSaving(true);
    setError("");
    try {
      const updated = await api.patch<Guild>(`/api/v1/guilds/${guild.id}`, {
        name: trimmed,
      });
      setGuild(updated);
      onClose();
    } catch {
      setError("Failed to update server");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    setDeleting(true);
    try {
      await api.delete(`/api/v1/guilds/${guild.id}`);
      removeGuild(guild.id);
      selectGuild(null);
      onClose();
    } catch {
      setError("Failed to delete server");
    } finally {
      setDeleting(false);
    }
  }

  async function handleLeave() {
    try {
      await api.delete(`/api/v1/guilds/${guild.id}/members/@me`);
      removeGuild(guild.id);
      selectGuild(null);
      onClose();
    } catch {
      setError("Failed to leave server");
    }
  }

  const tabs: { id: Tab; label: string }[] = [
    { id: "general", label: "General" },
    { id: "roles", label: "Roles" },
  ];

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70"
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg rounded-lg bg-bg-primary p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-4 text-xl font-bold text-text-primary">
          Server Settings
        </h2>

        {/* Tabs */}
        <div className="mb-4 flex gap-1 border-b border-border">
          {tabs.map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`px-3 py-2 text-sm font-medium transition-colors ${
                tab === t.id
                  ? "border-b-2 border-accent text-text-primary"
                  : "text-text-muted hover:text-text-secondary"
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>

        {tab === "general" && (
          <>
            {error && (
              <div className="mb-3 rounded bg-red-500/10 p-2 text-sm text-red-400">
                {error}
              </div>
            )}

            <form onSubmit={handleSave}>
              <label className="mb-2 block text-xs font-semibold uppercase text-text-secondary">
                Server Name
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="mb-4 w-full rounded bg-bg-input p-2.5 text-text-primary outline-none focus:ring-2 focus:ring-accent"
                disabled={!isOwner}
              />

              {isOwner && (
                <div className="mb-4 flex justify-end">
                  <button
                    type="submit"
                    disabled={saving || !name.trim()}
                    className="rounded bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent-hover disabled:opacity-50"
                  >
                    {saving ? "Saving..." : "Save Changes"}
                  </button>
                </div>
              )}
            </form>

            <div className="border-t border-border pt-4">
              {isOwner ? (
                <>
                  {!confirmDelete ? (
                    <button
                      onClick={() => setConfirmDelete(true)}
                      className="w-full rounded bg-red-500/20 px-4 py-2 text-sm font-medium text-red-400 hover:bg-red-500/30"
                    >
                      Delete Server
                    </button>
                  ) : (
                    <div className="space-y-2">
                      <p className="text-sm text-red-400">
                        Are you sure? This cannot be undone.
                      </p>
                      <div className="flex gap-2">
                        <button
                          onClick={handleDelete}
                          disabled={deleting}
                          className="flex-1 rounded bg-red-500 px-4 py-2 text-sm font-medium text-white hover:bg-red-600 disabled:opacity-50"
                        >
                          {deleting ? "Deleting..." : "Yes, Delete"}
                        </button>
                        <button
                          onClick={() => setConfirmDelete(false)}
                          className="flex-1 rounded bg-bg-secondary px-4 py-2 text-sm text-text-secondary hover:text-text-primary"
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <button
                  onClick={handleLeave}
                  className="w-full rounded bg-red-500/20 px-4 py-2 text-sm font-medium text-red-400 hover:bg-red-500/30"
                >
                  Leave Server
                </button>
              )}
            </div>
          </>
        )}

        {tab === "roles" && <RoleEditor guildId={guild.id} />}

        <div className="mt-4 flex justify-end">
          <button
            onClick={onClose}
            className="rounded px-4 py-2 text-sm text-text-secondary hover:text-text-primary"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
