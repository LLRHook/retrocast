import { useEffect, useState } from "react";
import { useMembersStore } from "@/stores/members";
import { useRolesStore } from "@/stores/roles";
import { usePresenceStore } from "@/stores/presence";
import { useUsersStore } from "@/stores/users";
import AvatarView from "@/components/AvatarView";
import UserProfilePopover from "@/components/UserProfilePopover";
import type { Member, Role } from "@/types";

interface MemberListProps {
  guildId: string;
}

interface MemberDisplayInfo {
  member: Member;
  displayName: string;
  username: string;
  avatarHash: string | null;
}

function intToHex(color: number): string {
  if (color === 0) return "";
  return `#${color.toString(16).padStart(6, "0")}`;
}

function MemberRow({
  info,
  guildId,
  highestRole,
}: {
  info: MemberDisplayInfo;
  guildId: string;
  highestRole: Role | null;
}) {
  const [popover, setPopover] = useState<{ x: number; y: number } | null>(
    null,
  );
  const status = usePresenceStore(
    (s) => s.presences.get(info.member.user_id) || "offline",
  );
  const isOffline = status === "offline" || status === "invisible";

  function handleClick(e: React.MouseEvent) {
    setPopover({ x: e.clientX - 310, y: e.clientY - 20 });
  }

  const nameColor = highestRole ? intToHex(highestRole.color) : "";

  return (
    <>
      <button
        onClick={handleClick}
        className={`flex w-full items-center gap-2 rounded px-2 py-1 text-left hover:bg-white/[0.04] ${
          isOffline ? "opacity-40" : ""
        }`}
      >
        <AvatarView
          userId={info.member.user_id}
          displayName={info.displayName}
          avatarHash={info.avatarHash}
          size="sm"
          showPresence
        />
        <span
          className="truncate text-sm font-medium"
          style={nameColor ? { color: nameColor } : undefined}
        >
          {info.displayName}
        </span>
      </button>
      {popover && (
        <UserProfilePopover
          member={info.member}
          displayName={info.displayName}
          username={info.username}
          avatarHash={info.avatarHash}
          guildId={guildId}
          x={popover.x}
          y={popover.y}
          onClose={() => setPopover(null)}
        />
      )}
    </>
  );
}

export default function MemberList({ guildId }: MemberListProps) {
  const membersMap = useMembersStore(
    (s) => s.membersByGuild.get(guildId) || new Map<string, Member>(),
  );
  const fetchMembers = useMembersStore((s) => s.fetchMembers);
  const roles = useRolesStore((s) => s.rolesByGuild.get(guildId) || []);
  const fetchRoles = useRolesStore((s) => s.fetchRoles);
  const presences = usePresenceStore((s) => s.presences);
  const usersCache = useUsersStore((s) => s.users);

  useEffect(() => {
    fetchMembers(guildId).catch(() => {});
    fetchRoles(guildId).catch(() => {});
  }, [guildId, fetchMembers, fetchRoles]);

  const members = Array.from(membersMap.values());

  // Build display info for each member, using the users cache for names/avatars
  const memberInfos: MemberDisplayInfo[] = members.map((m) => {
    const cached = usersCache.get(m.user_id);
    return {
      member: m,
      displayName: m.nickname || cached?.display_name || cached?.username || `User ${m.user_id.slice(-4)}`,
      username: cached?.username || m.user_id.slice(-8),
      avatarHash: cached?.avatar_hash ?? null,
    };
  });

  // Find highest role for each member (for coloring and grouping)
  function getHighestRole(member: Member): Role | null {
    const memberRoles = roles.filter(
      (r) => !r.is_default && member.roles.includes(r.id),
    );
    if (memberRoles.length === 0) return null;
    return memberRoles.reduce((highest, r) =>
      r.position > highest.position ? r : highest,
    );
  }

  // Group members by their highest role
  const roleGroups = new Map<string, MemberDisplayInfo[]>();
  const noRoleMembers: MemberDisplayInfo[] = [];

  for (const info of memberInfos) {
    const highest = getHighestRole(info.member);
    if (highest) {
      const group = roleGroups.get(highest.id) || [];
      group.push(info);
      roleGroups.set(highest.id, group);
    } else {
      noRoleMembers.push(info);
    }
  }

  // Sort members within groups: online first, then alphabetically
  function sortMembers(list: MemberDisplayInfo[]): MemberDisplayInfo[] {
    return [...list].sort((a, b) => {
      const aOnline = isOnline(a.member.user_id);
      const bOnline = isOnline(b.member.user_id);
      if (aOnline !== bOnline) return aOnline ? -1 : 1;
      return a.displayName.localeCompare(b.displayName);
    });
  }

  function isOnline(userId: string): boolean {
    const s = presences.get(userId);
    return s === "online" || s === "idle" || s === "dnd";
  }

  // Sorted role groups by position (highest first)
  const sortedRoles = roles
    .filter((r) => !r.is_default && roleGroups.has(r.id))
    .sort((a, b) => b.position - a.position);

  const onlineCount = members.filter((m) => isOnline(m.user_id)).length;

  return (
    <div className="flex w-60 shrink-0 flex-col overflow-y-auto border-l border-border bg-bg-secondary">
      <div className="p-3">
        <div className="mb-3 text-xs font-semibold uppercase text-text-muted">
          Members — {onlineCount} Online
        </div>

        {/* Role groups */}
        {sortedRoles.map((role) => {
          const group = sortMembers(roleGroups.get(role.id) || []);
          return (
            <div key={role.id} className="mb-3">
              <div className="mb-1 px-2 text-xs font-semibold uppercase text-text-muted">
                {role.name} — {group.length}
              </div>
              {group.map((info) => (
                <MemberRow
                  key={info.member.user_id}
                  info={info}
                  guildId={guildId}
                  highestRole={getHighestRole(info.member)}
                />
              ))}
            </div>
          );
        })}

        {/* Members with no special role — split into Online / Offline */}
        {(() => {
          const online = noRoleMembers.filter((i) => isOnline(i.member.user_id));
          const offline = noRoleMembers.filter((i) => !isOnline(i.member.user_id));
          return (
            <>
              {online.length > 0 && (
                <div className="mb-3">
                  <div className="mb-1 px-2 text-xs font-semibold uppercase text-text-muted">
                    Online — {online.length}
                  </div>
                  {sortMembers(online).map((info) => (
                    <MemberRow
                      key={info.member.user_id}
                      info={info}
                      guildId={guildId}
                      highestRole={null}
                    />
                  ))}
                </div>
              )}
              {offline.length > 0 && (
                <div className="mb-3">
                  <div className="mb-1 px-2 text-xs font-semibold uppercase text-text-muted">
                    Offline — {offline.length}
                  </div>
                  {sortMembers(offline).map((info) => (
                    <MemberRow
                      key={info.member.user_id}
                      info={info}
                      guildId={guildId}
                      highestRole={null}
                    />
                  ))}
                </div>
              )}
            </>
          );
        })()}
      </div>
    </div>
  );
}
