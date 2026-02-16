import { useState } from "react";
import { api } from "@/lib/api";
import { useChannelsStore } from "@/stores/channels";
import type { Channel } from "@/types";

interface ChannelModalProps {
  guildId: string;
  channel?: Channel; // If provided, editing; otherwise creating
  onClose: () => void;
}

export default function ChannelModal({
  guildId,
  channel,
  onClose,
}: ChannelModalProps) {
  const [name, setName] = useState(channel?.name || "");
  const [topic, setTopic] = useState(channel?.topic || "");
  const [type, setType] = useState(channel?.type ?? 0);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const addChannel = useChannelsStore((s) => s.addChannel);
  const updateChannel = useChannelsStore((s) => s.updateChannel);

  const isEditing = !!channel;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;

    setSaving(true);
    setError("");
    try {
      if (isEditing) {
        const updated = await api.patch<Channel>(
          `/api/v1/channels/${channel.id}`,
          { name: trimmed, topic: topic.trim() || null },
        );
        updateChannel(updated);
      } else {
        const created = await api.post<Channel>(
          `/api/v1/guilds/${guildId}/channels`,
          { name: trimmed, type, topic: topic.trim() || null },
        );
        addChannel(created);
      }
      onClose();
    } catch {
      setError(isEditing ? "Failed to update channel" : "Failed to create channel");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!channel) return;
    if (!confirm("Delete this channel? This cannot be undone.")) return;
    try {
      await api.delete(`/api/v1/channels/${channel.id}`);
      useChannelsStore.getState().removeChannel(channel.id, guildId);
      onClose();
    } catch {
      setError("Failed to delete channel");
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
          {isEditing ? "Edit Channel" : "Create Channel"}
        </h2>

        {error && (
          <div className="mb-3 rounded bg-red-500/10 p-2 text-sm text-red-400">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          {!isEditing && (
            <div className="mb-4">
              <label className="mb-2 block text-xs font-semibold uppercase text-text-secondary">
                Channel Type
              </label>
              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={() => setType(0)}
                  className={`flex-1 rounded p-2 text-sm ${
                    type === 0
                      ? "bg-accent text-white"
                      : "bg-bg-secondary text-text-secondary hover:text-text-primary"
                  }`}
                >
                  # Text
                </button>
                <button
                  type="button"
                  onClick={() => setType(2)}
                  className={`flex-1 rounded p-2 text-sm ${
                    type === 2
                      ? "bg-accent text-white"
                      : "bg-bg-secondary text-text-secondary hover:text-text-primary"
                  }`}
                >
                  Voice
                </button>
              </div>
            </div>
          )}

          <label className="mb-2 block text-xs font-semibold uppercase text-text-secondary">
            Channel Name
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="new-channel"
            className="mb-4 w-full rounded bg-bg-input p-2.5 text-text-primary outline-none focus:ring-2 focus:ring-accent"
            autoFocus
          />

          <label className="mb-2 block text-xs font-semibold uppercase text-text-secondary">
            Topic
          </label>
          <input
            type="text"
            value={topic}
            onChange={(e) => setTopic(e.target.value)}
            placeholder="What is this channel about?"
            className="mb-4 w-full rounded bg-bg-input p-2.5 text-text-primary outline-none focus:ring-2 focus:ring-accent"
          />

          <div className="flex justify-between">
            {isEditing && (
              <button
                type="button"
                onClick={handleDelete}
                className="rounded bg-red-500/20 px-4 py-2 text-sm text-red-400 hover:bg-red-500/30"
              >
                Delete Channel
              </button>
            )}
            <div className="ml-auto flex gap-3">
              <button
                type="button"
                onClick={onClose}
                className="rounded px-4 py-2 text-sm text-text-secondary hover:text-text-primary"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={saving || !name.trim()}
                className="rounded bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent-hover disabled:opacity-50"
              >
                {saving
                  ? "Saving..."
                  : isEditing
                    ? "Save Changes"
                    : "Create Channel"}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}
