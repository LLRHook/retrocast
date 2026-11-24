import type { Role } from "@/types";

// Append above roles the actor may manage, leaving their hierarchy boundary fixed.
export function nextRolePosition(roles: Role[], isOwner: boolean, highest: number): number | null {
  const top = Math.max(0, ...roles.filter(r => !r.is_default && (isOwner || r.position < highest)).map(r => r.position));
  const next = top + 1;
  return isOwner || next < highest ? next : null;
}
