import { useEffect, useRef, useState } from "react";
import { api, ApiClientError } from "@/lib/api";
import { useModeration } from "@/hooks/useModeration";
import { usePresenceStore } from "@/stores/presence";
import { useRolesStore } from "@/stores/roles";
import AvatarView from "@/components/AvatarView";
import type { Member } from "@/types";

interface UserProfilePopoverProps {
  member: Member;
  displayName: string;
  username: string;
  avatarHash: string | null;
  guildId: string;
  x: number;
  y: number;
  onClose: () => void;
}

function intToHex(color: number): string {
  if (color === 0) return "";
  return `#${color.toString(16).padStart(6, "0")}`;
}

export default function UserProfilePopover({
  member,
  displayName,
  username,
  avatarHash,
  guildId,
  x,
  y,
  onClose,
}: UserProfilePopoverProps) {
  const moderation = useModeration(guildId);
  const [action, setAction] = useState<"kick" | "ban" | null>(null);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function moderate() {
    setBusy(true); setError("");
    try {
      if (action === "ban") await api.put(`/api/v1/guilds/${guildId}/bans/${member.user_id}`, {reason: reason.trim() || undefined});
      else await api.delete(`/api/v1/guilds/${guildId}/members/${member.user_id}`);
      onClose();
    } catch (err) {
      setError(err instanceof ApiClientError && err.code === "ROLE_HIERARCHY"
        ? "This member ranks too high. Your highest role must be above theirs."
        : err instanceof ApiClientError ? err.message : "Could not remove member. Try again.");
    } finally { setBusy(false); }
  }
  const ref = useRef<HTMLDivElement>(null);
  const status = usePresenceStore(
    (s) => s.presences.get(member.user_id) || "offline",
  );
  const roles = (useRolesStore((s) => s.rolesByGuild.get(guildId)) || []);

  const memberRoles = roles.filter(
    (r) => !r.is_default && member.roles.includes(r.id),
  );

  const joinedAt = new Date(member.joined_at).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });

  // Close on click outside or Escape
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose();
      }
    }
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("mousedown", handleClick);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("keydown", handleKey);
    };
  }, [onClose]);

  // Position: try to keep within viewport
  const style: React.CSSProperties = {
    left: Math.min(x, window.innerWidth - 320),
    top: Math.max(8, Math.min(y, window.innerHeight - 550)),
    maxHeight: "calc(100vh - 16px)", overflowY: "auto",
  };

  return (
    <div
      ref={ref}
      className="fixed z-50 w-[300px] rounded-lg bg-bg-tertiary shadow-xl"
      style={style}
    >
      {/* Banner area */}
      <div className="h-16 rounded-t-lg bg-accent" />

      {/* Avatar overlapping banner */}
      <div className="relative px-4">
        <div className="-mt-10 mb-2">
          <AvatarView
            userId={member.user_id}
            displayName={displayName}
            avatarHash={avatarHash}
            size="lg"
            showPresence
          />
        </div>

        {/* Name */}
        <div className="mb-1">
          <div className="text-lg font-bold text-text-primary">
            {displayName}
          </div>
          <div className="text-sm text-text-secondary">{username}</div>
        </div>

        {/* Status */}
        <div className="mb-3 flex items-center gap-1.5 text-sm text-text-muted">
          <span
            className={`inline-block h-2 w-2 rounded-full ${
              status === "online"
                ? "bg-green-500"
                : status === "idle"
                  ? "bg-yellow-500"
                  : status === "dnd"
                    ? "bg-red-500"
                    : "bg-gray-500"
            }`}
          />
          {status === "dnd" ? "Do Not Disturb" : status.charAt(0).toUpperCase() + status.slice(1)}
        </div>

        <div className="border-t border-border pt-3" />

        {/* Roles */}
        {memberRoles.length > 0 && (
          <div className="mb-3">
            <div className="mb-1.5 text-xs font-semibold uppercase text-text-muted">
              Roles
            </div>
            <div className="flex flex-wrap gap-1">
              {memberRoles.map((role) => {
                const hex = intToHex(role.color);
                return (
                  <span
                    key={role.id}
                    className="inline-flex items-center gap-1 rounded bg-bg-secondary px-1.5 py-0.5 text-xs"
                  >
                    {hex && (
                      <span
                        className="inline-block h-2.5 w-2.5 rounded-full"
                        style={{ backgroundColor: hex }}
                      />
                    )}
                    <span className="text-text-secondary">{role.name}</span>
                  </span>
                );
              })}
            </div>
          </div>
        )}

        {moderation.userId !== member.user_id && (moderation.kick || moderation.ban) && (
          <div className="mb-4 border-t border-border pt-3">
            {!action ? <div className="flex gap-2">
              {moderation.kick && <button className="rounded bg-red-500/20 px-3 py-2 text-sm text-red-400" onClick={() => setAction("kick")}>Kick</button>}
              {moderation.ban && <button className="rounded bg-red-500/20 px-3 py-2 text-sm text-red-400" onClick={() => setAction("ban")}>Ban</button>}
            </div> : <div className="space-y-2">
              <p className="text-sm text-text-secondary">{action === "ban" ? "Ban" : "Kick"} {displayName} from this server?</p>
              {action === "ban" && <label className="block text-sm text-text-secondary">Reason (optional)<input value={reason} onChange={e => setReason(e.target.value)} className="mt-1 w-full rounded bg-bg-input p-2 text-text-primary" /></label>}
              {error && <p role="alert" className="text-sm text-red-400">{error}</p>}
              <div className="flex gap-2"><button disabled={busy} onClick={moderate} className="rounded bg-red-500 px-3 py-2 text-sm text-white disabled:opacity-50">{busy ? "Removing..." : action === "ban" ? "Confirm Ban" : "Confirm Kick"}</button>
              <button disabled={busy} onClick={() => {setAction(null); setError("");}} className="px-3 text-sm text-text-secondary">Cancel</button></div>
            </div>}
          </div>
        )}
        {/* Member since */}
        <div className="mb-4">
          <div className="mb-1 text-xs font-semibold uppercase text-text-muted">
            Member Since
          </div>
          <div className="text-sm text-text-secondary">{joinedAt}</div>
        </div>
      </div>
    </div>
  );
}
