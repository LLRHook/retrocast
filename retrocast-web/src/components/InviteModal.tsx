import { useState, useEffect } from "react";
import { api } from "@/lib/api";
import type { Invite } from "@/types";

interface InviteModalProps {
  guildId: string;
  onClose: () => void;
}

export default function InviteModal({ guildId, onClose }: InviteModalProps) {
  const [invites, setInvites] = useState<Invite[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    api
      .get<Invite[]>(`/api/v1/guilds/${guildId}/invites`)
      .then(setInvites)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [guildId]);

  async function handleCreate() {
    setCreating(true);
    try {
      const invite = await api.post<Invite>(
        `/api/v1/guilds/${guildId}/invites`,
        { max_uses: 0, max_age_seconds: 86400 * 7 },
      );
      setInvites((prev) => [invite, ...prev]);
    } catch {
      // ignore
    } finally {
      setCreating(false);
    }
  }

  async function handleRevoke(code: string) {
    try {
      await api.delete(`/invites/${code}`);
      setInvites((prev) => prev.filter((i) => i.code !== code));
    } catch {
      // ignore
    }
  }

  function handleCopy(code: string) {
    const url = `${window.location.origin}/invite/${code}`;
    navigator.clipboard.writeText(url).catch(() => {});
    setCopied(code);
    setTimeout(() => setCopied(null), 2000);
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
          Invite People
        </h2>

        <button
          onClick={handleCreate}
          disabled={creating}
          className="mb-4 w-full rounded bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent-hover disabled:opacity-50"
        >
          {creating ? "Creating..." : "Create Invite Link"}
        </button>

        {loading && (
          <div className="py-4 text-center text-sm text-text-muted">
            Loading invites...
          </div>
        )}

        {!loading && invites.length === 0 && (
          <div className="py-4 text-center text-sm text-text-muted">
            No active invites. Create one above.
          </div>
        )}

        <div className="max-h-64 space-y-2 overflow-y-auto">
          {invites.map((invite) => (
            <div
              key={invite.code}
              className="flex items-center justify-between rounded bg-bg-secondary p-3"
            >
              <div className="min-w-0 flex-1">
                <div className="font-mono text-sm text-text-primary">
                  {invite.code}
                </div>
                <div className="text-xs text-text-muted">
                  {invite.uses}
                  {invite.max_uses > 0 ? `/${invite.max_uses}` : ""} uses
                  {invite.expires_at &&
                    ` · Expires ${new Date(invite.expires_at).toLocaleDateString()}`}
                </div>
              </div>
              <div className="ml-2 flex gap-1">
                <button
                  onClick={() => handleCopy(invite.code)}
                  className="rounded bg-accent px-2 py-1 text-xs text-white hover:bg-accent-hover"
                >
                  {copied === invite.code ? "Copied!" : "Copy"}
                </button>
                <button
                  onClick={() => handleRevoke(invite.code)}
                  className="rounded bg-red-500/20 px-2 py-1 text-xs text-red-400 hover:bg-red-500/30"
                >
                  Revoke
                </button>
              </div>
            </div>
          ))}
        </div>

        <div className="mt-4 flex justify-end">
          <button
            onClick={onClose}
            className="rounded px-4 py-2 text-sm text-text-secondary hover:text-text-primary"
          >
            Done
          </button>
        </div>
      </div>
    </div>
  );
}
