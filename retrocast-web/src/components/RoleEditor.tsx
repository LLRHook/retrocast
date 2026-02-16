import { useState, useEffect } from "react";
import { api } from "@/lib/api";
import { useRolesStore } from "@/stores/roles";
import type { Role } from "@/types";

interface RoleEditorProps {
  guildId: string;
}

const PERMISSION_BITS: { name: string; bit: number; category: string }[] = [
  { name: "Administrator", bit: 1 << 31, category: "General" },
  { name: "Manage Guild", bit: 1 << 7, category: "General" },
  { name: "Manage Channels", bit: 1 << 3, category: "General" },
  { name: "Manage Roles", bit: 1 << 4, category: "General" },
  { name: "Create Invite", bit: 1 << 16, category: "General" },
  { name: "Change Nickname", bit: 1 << 17, category: "General" },
  { name: "Manage Nicknames", bit: 1 << 18, category: "General" },
  { name: "Kick Members", bit: 1 << 5, category: "Moderation" },
  { name: "Ban Members", bit: 1 << 6, category: "Moderation" },
  { name: "View Channel", bit: 1 << 0, category: "Text" },
  { name: "Send Messages", bit: 1 << 1, category: "Text" },
  { name: "Manage Messages", bit: 1 << 2, category: "Text" },
  { name: "Mention Everyone", bit: 1 << 13, category: "Text" },
  { name: "Attach Files", bit: 1 << 14, category: "Text" },
  { name: "Read Message History", bit: 1 << 15, category: "Text" },
  { name: "Connect", bit: 1 << 8, category: "Voice" },
  { name: "Speak", bit: 1 << 9, category: "Voice" },
  { name: "Mute Members", bit: 1 << 10, category: "Voice" },
  { name: "Deafen Members", bit: 1 << 11, category: "Voice" },
  { name: "Move Members", bit: 1 << 12, category: "Voice" },
];

function intToHexColor(color: number): string {
  return "#" + (color & 0xffffff).toString(16).padStart(6, "0");
}

function hexColorToInt(hex: string): number {
  return parseInt(hex.replace("#", ""), 16);
}

function PermissionToggle({
  name,
  enabled,
  onChange,
}: {
  name: string;
  enabled: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex cursor-pointer items-center justify-between py-1.5">
      <span className="text-sm text-text-secondary">{name}</span>
      <button
        type="button"
        onClick={() => onChange(!enabled)}
        className={`relative h-5 w-9 rounded-full transition-colors ${
          enabled ? "bg-accent" : "bg-bg-input"
        }`}
      >
        <span
          className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform ${
            enabled ? "left-[18px]" : "left-0.5"
          }`}
        />
      </button>
    </label>
  );
}

function RoleForm({
  role,
  guildId,
  onCancel,
}: {
  role: Role | null;
  guildId: string;
  onCancel: () => void;
}) {
  const [name, setName] = useState(role?.name || "");
  const [color, setColor] = useState(
    role?.color ? intToHexColor(role.color) : "#99aab5",
  );
  const [permissions, setPermissions] = useState(
    role ? Number(role.permissions) : 0,
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const setRole = useRolesStore((s) => s.setRole);

  useEffect(() => {
    setName(role?.name || "");
    setColor(role?.color ? intToHexColor(role.color) : "#99aab5");
    setPermissions(role ? Number(role.permissions) : 0);
    setError("");
  }, [role]);

  function togglePermission(bit: number) {
    setPermissions((prev) =>
      (prev & bit) !== 0 ? prev & ~bit : prev | bit,
    );
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;

    setSaving(true);
    setError("");
    try {
      const payload = {
        name: trimmed,
        color: hexColorToInt(color),
        permissions: String(permissions),
      };

      if (role) {
        const updated = await api.patch<Role>(
          `/api/v1/guilds/${guildId}/roles/${role.id}`,
          payload,
        );
        setRole(updated);
      } else {
        const created = await api.post<Role>(
          `/api/v1/guilds/${guildId}/roles`,
          payload,
        );
        setRole(created);
      }
      onCancel();
    } catch {
      setError(role ? "Failed to update role" : "Failed to create role");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!role || role.is_default) return;
    if (!confirm("Delete this role? This cannot be undone.")) return;
    try {
      await api.delete(`/api/v1/guilds/${guildId}/roles/${role.id}`);
      useRolesStore.getState().removeRole(guildId, role.id);
      onCancel();
    } catch {
      setError("Failed to delete role");
    }
  }

  const categories = [...new Set(PERMISSION_BITS.map((p) => p.category))];

  return (
    <form onSubmit={handleSave}>
      {error && (
        <div className="mb-3 rounded bg-red-500/10 p-2 text-sm text-red-400">
          {error}
        </div>
      )}

      <label className="mb-2 block text-xs font-semibold uppercase text-text-secondary">
        Role Name
      </label>
      <input
        type="text"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="New Role"
        className="mb-4 w-full rounded bg-bg-input p-2.5 text-text-primary outline-none focus:ring-2 focus:ring-accent"
        disabled={role?.is_default}
      />

      <label className="mb-2 block text-xs font-semibold uppercase text-text-secondary">
        Color
      </label>
      <div className="mb-4 flex items-center gap-3">
        <input
          type="color"
          value={color}
          onChange={(e) => setColor(e.target.value)}
          className="h-8 w-8 cursor-pointer rounded border-0 bg-transparent"
        />
        <span className="text-sm text-text-muted">{color}</span>
      </div>

      <div className="mb-4 max-h-64 overflow-y-auto">
        {categories.map((cat) => (
          <div key={cat} className="mb-3">
            <div className="mb-1 text-xs font-semibold uppercase text-text-muted">
              {cat}
            </div>
            {PERMISSION_BITS.filter((p) => p.category === cat).map((perm) => (
              <PermissionToggle
                key={perm.bit}
                name={perm.name}
                enabled={(permissions & perm.bit) !== 0}
                onChange={() => togglePermission(perm.bit)}
              />
            ))}
          </div>
        ))}
      </div>

      <div className="flex justify-between">
        {role && !role.is_default && (
          <button
            type="button"
            onClick={handleDelete}
            className="rounded bg-red-500/20 px-3 py-1.5 text-sm text-red-400 hover:bg-red-500/30"
          >
            Delete
          </button>
        )}
        <div className="ml-auto flex gap-2">
          <button
            type="button"
            onClick={onCancel}
            className="rounded px-3 py-1.5 text-sm text-text-secondary hover:text-text-primary"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={saving || !name.trim()}
            className="rounded bg-accent px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-hover disabled:opacity-50"
          >
            {saving ? "Saving..." : role ? "Save" : "Create"}
          </button>
        </div>
      </div>
    </form>
  );
}

export default function RoleEditor({ guildId }: RoleEditorProps) {
  const roles = useRolesStore((s) => s.rolesByGuild.get(guildId) || []);
  const fetchRoles = useRolesStore((s) => s.fetchRoles);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    fetchRoles(guildId);
  }, [guildId, fetchRoles]);

  if (creating || selectedRole) {
    return (
      <div>
        <button
          onClick={() => {
            setSelectedRole(null);
            setCreating(false);
          }}
          className="mb-3 flex items-center gap-1 text-sm text-text-muted hover:text-text-primary"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2Z" />
          </svg>
          Back to roles
        </button>
        <RoleForm
          role={selectedRole}
          guildId={guildId}
          onCancel={() => {
            setSelectedRole(null);
            setCreating(false);
          }}
        />
      </div>
    );
  }

  return (
    <div>
      <div className="mb-3 flex items-center justify-between">
        <span className="text-xs font-semibold uppercase text-text-secondary">
          Roles ({roles.length})
        </span>
        <button
          onClick={() => setCreating(true)}
          className="rounded bg-accent px-2.5 py-1 text-xs font-medium text-white hover:bg-accent-hover"
        >
          Create Role
        </button>
      </div>
      <div className="flex flex-col gap-1">
        {roles.map((role) => (
          <button
            key={role.id}
            onClick={() => setSelectedRole(role)}
            className="flex items-center gap-2 rounded px-2 py-1.5 text-left text-sm transition-colors hover:bg-white/5"
          >
            <span
              className="h-3 w-3 shrink-0 rounded-full"
              style={{
                backgroundColor:
                  role.color ? intToHexColor(role.color) : "#99aab5",
              }}
            />
            <span className="truncate text-text-secondary">{role.name}</span>
            {role.is_default && (
              <span className="ml-auto text-xs text-text-muted">default</span>
            )}
          </button>
        ))}
        {roles.length === 0 && (
          <div className="text-center text-sm text-text-muted">
            No roles yet
          </div>
        )}
      </div>
    </div>
  );
}
