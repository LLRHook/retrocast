import { useState } from "react";
import { useChannelsStore } from "@/stores/channels";
import { useGuildsStore } from "@/stores/guilds";
import { useDMsStore } from "@/stores/dms";
import { useAuthStore } from "@/stores/auth";
import MessageList from "@/components/MessageList";
import MessageInput from "@/components/MessageInput";
import TypingIndicator from "@/components/TypingIndicator";
import FileUpload from "@/components/FileUpload";
import MemberList from "@/components/MemberList";
import AvatarView from "@/components/AvatarView";

export default function ChatArea() {
  const selectedChannelId = useChannelsStore((s) => s.selectedChannelId);
  const selectedGuildId = useGuildsStore((s) => s.selectedGuildId);
  const channelsByGuild = useChannelsStore((s) => s.channelsByGuild);
  const showDMList = useDMsStore((s) => s.showDMList);
  const selectedDMId = useDMsStore((s) => s.selectedDMId);
  const dms = useDMsStore((s) => s.dms);
  const currentUser = useAuthStore((s) => s.user);
  const [showMembers, setShowMembers] = useState(true);
  const [pendingDropFiles, setPendingDropFiles] = useState<File[]>([]);

  // Determine which channel to show
  const isDM = showDMList && selectedDMId;
  const activeChannelId = isDM ? selectedDMId : selectedChannelId;

  if (!activeChannelId) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="text-center">
          <h1 className="mb-2 text-2xl font-bold text-text-primary">
            Welcome to Retrocast
          </h1>
          <p className="text-text-secondary">
            Select or create a server to get started
          </p>
        </div>
      </div>
    );
  }

  // Header info depends on whether this is a DM or guild channel
  let headerContent: React.ReactNode;

  if (isDM) {
    const dm = dms.get(selectedDMId);
    const recipient = dm?.recipients.find((r) => r.id !== currentUser?.id) || dm?.recipients[0];
    const recipientName = recipient
      ? recipient.display_name || recipient.username
      : "Direct Message";

    headerContent = (
      <>
        {recipient && (
          <AvatarView
            userId={recipient.id}
            displayName={recipientName}
            avatarHash={recipient.avatar_hash}
            size="sm"
            showPresence
            className="mr-2"
          />
        )}
        <span className="font-semibold text-text-primary">{recipientName}</span>
      </>
    );
  } else {
    const channels = selectedGuildId
      ? channelsByGuild.get(selectedGuildId)
      : null;
    const channel = channels?.find((c) => c.id === selectedChannelId);

    headerContent = (
      <>
        <span className="mr-1 text-text-muted">#</span>
        <span className="font-semibold text-text-primary">
          {channel?.name || "channel"}
        </span>
        {channel?.topic && (
          <>
            <div className="mx-3 h-6 w-px bg-border" />
            <span className="truncate text-sm text-text-muted">
              {channel.topic}
            </span>
          </>
        )}
      </>
    );
  }

  // Channel name for MessageInput placeholder
  const channelName = isDM
    ? undefined
    : (() => {
        const channels = selectedGuildId
          ? channelsByGuild.get(selectedGuildId)
          : null;
        return channels?.find((c) => c.id === selectedChannelId)?.name;
      })();

  return (
    <div className="flex flex-1 overflow-hidden">
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Header */}
        <div className="flex h-12 shrink-0 items-center border-b border-border px-4">
          {headerContent}

          {/* Spacer */}
          <div className="flex-1" />

          {/* Member list toggle (guild channels only) */}
          {selectedGuildId && !isDM && (
            <button
              onClick={() => setShowMembers((v) => !v)}
              className={`rounded p-1.5 ${
                showMembers
                  ? "text-text-primary"
                  : "text-text-muted hover:text-text-secondary"
              }`}
              title={showMembers ? "Hide member list" : "Show member list"}
            >
              <svg
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="currentColor"
              >
                <path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3Zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3Zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5Zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5Z" />
              </svg>
            </button>
          )}
        </div>

        <FileUpload onFilesSelected={(files) => setPendingDropFiles(files)}>
          <MessageList channelId={activeChannelId} />
          <TypingIndicator channelId={activeChannelId} />
          <MessageInput
            channelId={activeChannelId}
            channelName={channelName}
            droppedFiles={pendingDropFiles}
            onDropConsumed={() => setPendingDropFiles([])}
          />
        </FileUpload>
      </div>

      {/* Member list panel (guild channels only) */}
      {showMembers && selectedGuildId && !isDM && (
        <MemberList guildId={selectedGuildId} />
      )}
    </div>
  );
}
