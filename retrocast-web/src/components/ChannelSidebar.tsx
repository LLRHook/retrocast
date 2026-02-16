import { useState } from "react";
import { useGuildsStore } from "@/stores/guilds";
import { useChannelsStore } from "@/stores/channels";
import { useDMsStore } from "@/stores/dms";
import { useAuthStore } from "@/stores/auth";
import AvatarView from "@/components/AvatarView";
import InviteModal from "@/components/InviteModal";
import GuildSettingsModal from "@/components/GuildSettingsModal";
import ChannelModal from "@/components/ChannelModal";
import UserSettingsModal from "@/components/UserSettingsModal";
import type { Channel } from "@/types";

const CHANNEL_TYPE_VOICE = 2;
const CHANNEL_TYPE_CATEGORY = 4;

function HashIcon() {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="currentColor"
      className="mr-1.5 shrink-0 opacity-60"
    >
      <path d="M5.88657 21.0001H7.88657L8.40657 18.0001H10.4066L9.88657 21.0001H11.8866L12.4066 18.0001H15.4066V16.0001H12.7466L13.2666 13.0001H16.2666V11.0001H13.6066L14.1266 8.00012H12.1266L11.6066 11.0001H9.60657L10.1266 8.00012H8.12657L7.60657 11.0001H4.60657V13.0001H7.26657L6.74657 16.0001H3.74657V18.0001H6.40657L5.88657 21.0001ZM9.26657 16.0001L9.74657 13.0001H11.7466L11.2666 16.0001H9.26657Z" />
    </svg>
  );
}

function VoiceIcon() {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="currentColor"
      className="mr-1.5 shrink-0 opacity-60"
    >
      <path d="M12 3a1 1 0 0 0-1 1v8a1 1 0 0 0 2 0V4a1 1 0 0 0-1-1ZM6.5 8a.5.5 0 0 0-1 0v4a6.5 6.5 0 0 0 12 3.25.5.5 0 1 0-.87-.5A5.5 5.5 0 0 1 6.5 12V8ZM11 19.93A5.51 5.51 0 0 1 6.5 14.5a.5.5 0 0 0-1 0A6.51 6.51 0 0 0 11 20.94V23a1 1 0 1 0 2 0v-2.06a6.51 6.51 0 0 0 5.5-6.44.5.5 0 0 0-1 0 5.51 5.51 0 0 1-4.5 5.43V19.93Z" />
    </svg>
  );
}

function ChannelItem({
  channel,
  isSelected,
  onClick,
  onEdit,
}: {
  channel: Channel;
  isSelected: boolean;
  onClick: () => void;
  onEdit?: () => void;
}) {
  return (
    <div className="group relative">
      <button
        onClick={onClick}
        className={`flex w-full items-center rounded px-2 py-1 text-left text-sm transition-colors ${
          isSelected
            ? "bg-white/10 text-text-primary"
            : "text-text-muted hover:bg-white/5 hover:text-text-secondary"
        }`}
      >
        {channel.type === CHANNEL_TYPE_VOICE ? <VoiceIcon /> : <HashIcon />}
        <span className="truncate">{channel.name}</span>
      </button>
      {onEdit && (
        <button
          onClick={onEdit}
          className="absolute right-1 top-1/2 hidden -translate-y-1/2 rounded p-0.5 text-text-muted hover:text-text-primary group-hover:block"
          title="Edit channel"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
            <path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58a.49.49 0 0 0 .12-.61l-1.92-3.32a.49.49 0 0 0-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.48.48 0 0 0-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.49.49 0 0 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.07.62-.07.94s.02.64.07.94l-2.03 1.58a.49.49 0 0 0-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58ZM12 15.6A3.6 3.6 0 1 1 12 8.4a3.6 3.6 0 0 1 0 7.2Z" />
          </svg>
        </button>
      )}
    </div>
  );
}

function CategoryGroup({
  name,
  channels,
  selectedChannelId,
  onSelectChannel,
  onEditChannel,
}: {
  name: string;
  channels: Channel[];
  selectedChannelId: string | null;
  onSelectChannel: (id: string) => void;
  onEditChannel: (channel: Channel) => void;
}) {
  return (
    <div className="mb-1">
      <div className="flex items-center px-1 pb-0.5 pt-4">
        <svg
          width="12"
          height="12"
          viewBox="0 0 24 24"
          fill="currentColor"
          className="mr-0.5 text-text-muted"
        >
          <path d="m5.7 10 5.3 5.3 5.3-5.3H5.7Z" />
        </svg>
        <span className="text-xs font-semibold uppercase tracking-wide text-text-muted">
          {name}
        </span>
      </div>
      <div className="ml-1 flex flex-col gap-0.5">
        {channels.map((ch) => (
          <ChannelItem
            key={ch.id}
            channel={ch}
            isSelected={selectedChannelId === ch.id}
            onClick={() => onSelectChannel(ch.id)}
            onEdit={() => onEditChannel(ch)}
          />
        ))}
      </div>
    </div>
  );
}

function UserPanel() {
  const user = useAuthStore((s) => s.user);
  const [showUserSettings, setShowUserSettings] = useState(false);

  if (!user) return null;

  const displayName = user.display_name || user.username;

  return (
    <>
      <div className="flex items-center gap-2 border-t border-border bg-bg-tertiary/50 px-2 py-2">
        <AvatarView
          userId={user.id}
          displayName={displayName}
          avatarHash={user.avatar_hash}
          size="md"
          showPresence
        />
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-text-primary">
            {displayName}
          </div>
          <div className="truncate text-xs text-text-muted">
            {user.username}
          </div>
        </div>
        <button
          onClick={() => setShowUserSettings(true)}
          className="shrink-0 rounded p-1 text-text-muted hover:text-text-primary"
          title="User Settings"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
            <path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58a.49.49 0 0 0 .12-.61l-1.92-3.32a.49.49 0 0 0-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.48.48 0 0 0-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.49.49 0 0 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.07.62-.07.94s.02.64.07.94l-2.03 1.58a.49.49 0 0 0-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58ZM12 15.6A3.6 3.6 0 1 1 12 8.4a3.6 3.6 0 0 1 0 7.2Z" />
          </svg>
        </button>
      </div>
      {showUserSettings && (
        <UserSettingsModal onClose={() => setShowUserSettings(false)} />
      )}
    </>
  );
}

function DMList() {
  const dms = useDMsStore((s) => s.dms);
  const selectedDMId = useDMsStore((s) => s.selectedDMId);
  const selectDM = useDMsStore((s) => s.selectDM);
  const currentUser = useAuthStore((s) => s.user);
  const dmList = Array.from(dms.values());

  return (
    <div className="flex w-60 shrink-0 flex-col bg-bg-secondary">
      <div className="flex h-12 items-center border-b border-border px-4">
        <span className="font-semibold text-text-primary">
          Direct Messages
        </span>
      </div>
      <div className="flex-1 overflow-y-auto px-2 py-1">
        {dmList.length === 0 && (
          <div className="flex items-center justify-center p-4 text-sm text-text-muted">
            No conversations yet
          </div>
        )}
        {dmList.map((dm) => {
          const recipient = dm.recipients.find((r) => r.id !== currentUser?.id) || dm.recipients[0];
          const name = recipient
            ? recipient.display_name || recipient.username
            : "Unknown";
          return (
            <button
              key={dm.id}
              onClick={() => selectDM(dm.id)}
              className={`flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm transition-colors ${
                selectedDMId === dm.id
                  ? "bg-white/10 text-text-primary"
                  : "text-text-muted hover:bg-white/5 hover:text-text-secondary"
              }`}
            >
              {recipient ? (
                <AvatarView
                  userId={recipient.id}
                  displayName={name}
                  avatarHash={recipient.avatar_hash}
                  size="md"
                  showPresence
                />
              ) : (
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-accent text-xs font-medium text-white">
                  ?
                </div>
              )}
              <span className="truncate">{name}</span>
            </button>
          );
        })}
      </div>
      <UserPanel />
    </div>
  );
}

export default function ChannelSidebar() {
  const showDMList = useDMsStore((s) => s.showDMList);
  const selectedGuildId = useGuildsStore((s) => s.selectedGuildId);
  const guild = useGuildsStore(
    (s) => (selectedGuildId ? s.guilds.get(selectedGuildId) : null),
  );
  const channelsByGuild = useChannelsStore((s) => s.channelsByGuild);
  const selectedChannelId = useChannelsStore((s) => s.selectedChannelId);
  const selectChannel = useChannelsStore((s) => s.selectChannel);

  const [showInvite, setShowInvite] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [showCreateChannel, setShowCreateChannel] = useState(false);
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null);

  if (showDMList) {
    return <DMList />;
  }

  if (!selectedGuildId || !guild) {
    return (
      <div className="flex w-60 shrink-0 flex-col bg-bg-secondary">
        <div className="flex h-12 items-center border-b border-border px-4">
          <span className="font-semibold text-text-primary">Retrocast</span>
        </div>
        <div className="flex flex-1 items-center justify-center p-4 text-sm text-text-muted">
          Select a server
        </div>
        <UserPanel />
      </div>
    );
  }

  const channels = channelsByGuild.get(selectedGuildId) || [];

  // Group channels: categories contain their children, ungrouped at top
  const categories = channels.filter((c) => c.type === CHANNEL_TYPE_CATEGORY);
  const ungrouped = channels.filter(
    (c) => c.type !== CHANNEL_TYPE_CATEGORY && c.parent_id === null,
  );

  return (
    <div className="flex w-60 shrink-0 flex-col bg-bg-secondary">
      {/* Guild header with actions */}
      <div className="flex h-12 items-center justify-between border-b border-border px-4">
        <span className="truncate font-semibold text-text-primary">
          {guild.name}
        </span>
        <div className="flex gap-1">
          <button
            onClick={() => setShowInvite(true)}
            className="rounded p-1 text-text-muted hover:text-text-primary"
            title="Invite People"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
              <path d="M15 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4Zm-9-2V7H4v3H1v2h3v3h2v-3h3v-2H6Zm9 4c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4Z" />
            </svg>
          </button>
          <button
            onClick={() => setShowSettings(true)}
            className="rounded p-1 text-text-muted hover:text-text-primary"
            title="Server Settings"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58a.49.49 0 0 0 .12-.61l-1.92-3.32a.49.49 0 0 0-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.48.48 0 0 0-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.49.49 0 0 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.07.62-.07.94s.02.64.07.94l-2.03 1.58a.49.49 0 0 0-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58ZM12 15.6A3.6 3.6 0 1 1 12 8.4a3.6 3.6 0 0 1 0 7.2Z" />
            </svg>
          </button>
        </div>
      </div>

      {/* Channel list */}
      <div className="flex-1 overflow-y-auto px-2 py-1">
        {/* Ungrouped channels */}
        {ungrouped.length > 0 && (
          <div className="mb-1 flex flex-col gap-0.5">
            {ungrouped.map((ch) => (
              <ChannelItem
                key={ch.id}
                channel={ch}
                isSelected={selectedChannelId === ch.id}
                onClick={() => selectChannel(ch.id)}
                onEdit={() => setEditingChannel(ch)}
              />
            ))}
          </div>
        )}

        {/* Category groups */}
        {categories.map((cat) => {
          const children = channels.filter(
            (c) => c.parent_id === cat.id && c.type !== CHANNEL_TYPE_CATEGORY,
          );
          return (
            <CategoryGroup
              key={cat.id}
              name={cat.name}
              channels={children}
              selectedChannelId={selectedChannelId}
              onSelectChannel={selectChannel}
              onEditChannel={setEditingChannel}
            />
          );
        })}
      </div>

      {/* Create channel button */}
      <div className="border-t border-border p-2">
        <button
          onClick={() => setShowCreateChannel(true)}
          className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-sm text-text-muted hover:bg-white/5 hover:text-text-secondary"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
            <path d="M13 5a1 1 0 1 0-2 0v6H5a1 1 0 1 0 0 2h6v6a1 1 0 1 0 2 0v-6h6a1 1 0 1 0 0-2h-6V5Z" />
          </svg>
          Create Channel
        </button>
      </div>

      <UserPanel />

      {/* Modals */}
      {showInvite && (
        <InviteModal
          guildId={selectedGuildId}
          onClose={() => setShowInvite(false)}
        />
      )}
      {showSettings && guild && (
        <GuildSettingsModal
          guild={guild}
          onClose={() => setShowSettings(false)}
        />
      )}
      {showCreateChannel && (
        <ChannelModal
          guildId={selectedGuildId}
          onClose={() => setShowCreateChannel(false)}
        />
      )}
      {editingChannel && (
        <ChannelModal
          guildId={selectedGuildId}
          channel={editingChannel}
          onClose={() => setEditingChannel(null)}
        />
      )}
    </div>
  );
}
