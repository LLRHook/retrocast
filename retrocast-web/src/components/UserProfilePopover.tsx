import { useEffect, useRef } from "react";
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
  const ref = useRef<HTMLDivElement>(null);
  const status = usePresenceStore(
    (s) => s.presences.get(member.user_id) || "offline",
  );
  const roles = useRolesStore((s) => s.rolesByGuild.get(guildId) || []);

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
    top: Math.min(y, window.innerHeight - 350),
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
