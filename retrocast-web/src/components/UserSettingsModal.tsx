import { useState } from "react";
import { api, ApiClientError } from "@/lib/api";
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

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [passwordMessage, setPasswordMessage] = useState("");
  const [changingPassword, setChangingPassword] = useState(false);
  async function changePassword(e: React.FormEvent) {
    e.preventDefault();
    if (newPassword !== confirmation) { setPasswordMessage("New passwords do not match"); return; }
    setChangingPassword(true);
    setPasswordMessage("");
    try {
      await useAuthStore.getState().changePassword(currentPassword, newPassword);
      setCurrentPassword(""); setNewPassword(""); setConfirmation("");
      setPasswordMessage("Password changed successfully. Other sessions have been signed out.");
    } catch (err) {
      setPasswordMessage(err instanceof ApiClientError && err.code === "INVALID_CREDENTIALS" ? "Current password is incorrect" : err instanceof Error ? err.message : "Could not change password");
    } finally { setChangingPassword(false); }
  }

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
    try {
      try {
        gateway.disconnect();
      } finally {
        await useAuthStore.getState().logout();
      }
    } catch {
      console.warn("Could not finish server sign-out");
    } finally {
      window.location.href = "/server";
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

        <form onSubmit={changePassword} className="mb-4 border-t border-border pt-4">
          <h3 className="mb-3 font-bold text-text-primary">Password</h3>
          {passwordMessage && <p role="status" className="mb-3 text-sm text-text-primary">{passwordMessage}</p>}
          <label htmlFor="current-password" className="block text-text-secondary">Current password</label>
          <input id="current-password" type="password" autoComplete="current-password" required value={currentPassword} onChange={e => setCurrentPassword(e.target.value)} className="mb-3 w-full rounded bg-bg-input p-2 text-text-primary" />
          <label htmlFor="new-password" className="block text-text-secondary">New password</label>
          <input id="new-password" type="password" autoComplete="new-password" required minLength={6} maxLength={128} value={newPassword} onChange={e => setNewPassword(e.target.value)} className="mb-3 w-full rounded bg-bg-input p-2 text-text-primary" />
          <label htmlFor="confirm-password" className="block text-text-secondary">Confirm new password</label>
          <input id="confirm-password" type="password" autoComplete="new-password" required value={confirmation} onChange={e => setConfirmation(e.target.value)} className="mb-3 w-full rounded bg-bg-input p-2 text-text-primary" />
          <button disabled={changingPassword} className="rounded bg-accent px-4 py-2 text-white disabled:opacity-50">{changingPassword ? "Changing..." : "Change Password"}</button>
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
