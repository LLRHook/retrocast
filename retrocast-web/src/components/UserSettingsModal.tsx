import { useState } from "react";
import { api } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import { gateway } from "@/lib/gateway";
import type { User } from "@/types";

interface UserSettingsModalProps {
  onClose: () => void;
}

export default function UserSettingsModal({ onClose }: UserSettingsModalProps) {
  const user = useAuthStore((s) => s.user);
  const [displayName, setDisplayName] = useState(user?.display_name || "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);

  if (!user) return null;

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = displayName.trim();
    if (!trimmed) return;

    setSaving(true);
    setError("");
    setSuccess(false);
    try {
      const updated = await api.patch<User>("/api/v1/users/@me", {
        display_name: trimmed,
      });
      useAuthStore.setState({ user: updated });
      setSuccess(true);
      setTimeout(() => setSuccess(false), 2000);
    } catch {
      setError("Failed to update profile");
    } finally {
      setSaving(false);
    }
  }

  async function handleLogout() {
    gateway.disconnect();
    await useAuthStore.getState().logout();
    window.location.href = "/server";
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
          User Settings
        </h2>

        {error && (
          <div className="mb-3 rounded bg-red-500/10 p-2 text-sm text-red-400">
            {error}
          </div>
        )}

        {success && (
          <div className="mb-3 rounded bg-green-500/10 p-2 text-sm text-green-400">
            Profile updated successfully
          </div>
        )}

        <form onSubmit={handleSave}>
          <label className="mb-2 block text-xs font-semibold uppercase text-text-secondary">
            Username
          </label>
          <input
            type="text"
            value={user.username}
            disabled
            className="mb-4 w-full rounded bg-bg-input p-2.5 text-text-muted outline-none"
          />

          <label className="mb-2 block text-xs font-semibold uppercase text-text-secondary">
            Display Name
          </label>
          <input
            type="text"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            className="mb-4 w-full rounded bg-bg-input p-2.5 text-text-primary outline-none focus:ring-2 focus:ring-accent"
            autoFocus
          />

          <div className="mb-4 flex justify-end">
            <button
              type="submit"
              disabled={saving || !displayName.trim()}
              className="rounded bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent-hover disabled:opacity-50"
            >
              {saving ? "Saving..." : "Save Changes"}
            </button>
          </div>
        </form>

        <div className="border-t border-border pt-4">
          <button
            onClick={handleLogout}
            className="w-full rounded bg-red-500/20 px-4 py-2 text-sm font-medium text-red-400 hover:bg-red-500/30"
          >
            Log Out
          </button>
        </div>

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
