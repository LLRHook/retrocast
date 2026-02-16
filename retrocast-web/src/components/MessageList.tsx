import { useEffect, useRef, useCallback, useState } from "react";
import { useMessagesStore } from "@/stores/messages";
import { useGuildsStore } from "@/stores/guilds";
import { useMembersStore } from "@/stores/members";
import { api } from "@/lib/api";
import type { Message as MessageType, Attachment } from "@/types";
import MarkdownContent from "@/components/MarkdownContent";
import MessageContextMenu from "@/components/MessageContextMenu";
import AvatarView from "@/components/AvatarView";
import UserProfilePopover from "@/components/UserProfilePopover";

function formatTimestamp(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const isToday = date.toDateString() === now.toDateString();
  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  const isYesterday = date.toDateString() === yesterday.toDateString();

  const time = date.toLocaleTimeString([], {
    hour: "numeric",
    minute: "2-digit",
  });

  if (isToday) return `Today at ${time}`;
  if (isYesterday) return `Yesterday at ${time}`;
  return `${date.toLocaleDateString()} ${time}`;
}

function formatDateSeparator(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const isToday = date.toDateString() === now.toDateString();
  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  const isYesterday = date.toDateString() === yesterday.toDateString();

  if (isToday) return "Today";
  if (isYesterday) return "Yesterday";
  return date.toLocaleDateString(undefined, {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function shouldGroup(prev: MessageType, curr: MessageType): boolean {
  if (prev.author_id !== curr.author_id) return false;
  const diff =
    new Date(curr.created_at).getTime() -
    new Date(prev.created_at).getTime();
  return diff < 5 * 60 * 1000;
}

function isDifferentDay(a: string, b: string): boolean {
  return new Date(a).toDateString() !== new Date(b).toDateString();
}

function AttachmentView({ attachment }: { attachment: Attachment }) {
  const serverUrl = localStorage.getItem("serverUrl") || "";
  const isImage = attachment.content_type.startsWith("image/");
  const url = attachment.url.startsWith("http")
    ? attachment.url
    : `${serverUrl}${attachment.url}`;

  if (isImage) {
    return (
      <a
        href={url}
        target="_blank"
        rel="noopener noreferrer"
        className="mt-1 block"
      >
        <img
          src={url}
          alt={attachment.filename}
          className="max-h-80 max-w-md rounded"
          loading="lazy"
        />
      </a>
    );
  }

  const sizeStr =
    attachment.size < 1024
      ? `${attachment.size} B`
      : attachment.size < 1024 * 1024
        ? `${(attachment.size / 1024).toFixed(1)} KB`
        : `${(attachment.size / (1024 * 1024)).toFixed(1)} MB`;

  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      className="mt-1 flex items-center gap-2 rounded border border-border bg-bg-secondary p-2 text-sm hover:bg-white/5"
    >
      <svg
        width="20"
        height="20"
        viewBox="0 0 24 24"
        fill="currentColor"
        className="shrink-0 text-text-muted"
      >
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6Zm4 18H6V4h7v5h5v11Z" />
      </svg>
      <div className="min-w-0">
        <div className="truncate text-accent">{attachment.filename}</div>
        <div className="text-xs text-text-muted">{sizeStr}</div>
      </div>
    </a>
  );
}

function DateSeparator({ date }: { date: string }) {
  return (
    <div className="my-2 flex items-center px-4">
      <div className="flex-1 border-t border-border" />
      <span className="mx-4 text-xs font-semibold text-text-muted">
        {formatDateSeparator(date)}
      </span>
      <div className="flex-1 border-t border-border" />
    </div>
  );
}

function MessageItem({
  message,
  grouped,
}: {
  message: MessageType;
  grouped: boolean;
}) {
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
  } | null>(null);
  const [editing, setEditing] = useState(false);
  const [editContent, setEditContent] = useState(message.content);
  const [saving, setSaving] = useState(false);
  const [profilePopover, setProfilePopover] = useState<{
    x: number;
    y: number;
  } | null>(null);

  const displayName =
    message.author_display_name || message.author_username;
  const selectedGuildId = useGuildsStore((s) => s.selectedGuildId);
  const member = useMembersStore((s) => {
    if (!selectedGuildId) return null;
    const guildMembers = s.membersByGuild.get(selectedGuildId);
    return guildMembers?.get(message.author_id) ?? null;
  });

  function handleNameClick(e: React.MouseEvent) {
    if (!selectedGuildId || !member) return;
    setProfilePopover({ x: e.clientX, y: e.clientY });
  }

  function handleContextMenu(e: React.MouseEvent) {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY });
  }

  async function handleSaveEdit() {
    const trimmed = editContent.trim();
    if (!trimmed || trimmed === message.content) {
      setEditing(false);
      return;
    }
    setSaving(true);
    try {
      await api.patch(
        `/api/v1/channels/${message.channel_id}/messages/${message.id}`,
        { content: trimmed },
      );
      setEditing(false);
    } catch {
      // ignore
    } finally {
      setSaving(false);
    }
  }

  function handleEditKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSaveEdit();
    }
    if (e.key === "Escape") {
      setEditing(false);
      setEditContent(message.content);
    }
  }

  const editArea = (
    <div>
      <textarea
        value={editContent}
        onChange={(e) => setEditContent(e.target.value)}
        onKeyDown={handleEditKeyDown}
        className="w-full rounded bg-bg-input p-2 text-sm text-text-primary outline-none"
        autoFocus
        rows={2}
      />
      <div className="mt-1 text-xs text-text-muted">
        Enter to save, Escape to cancel
        {saving && " — Saving..."}
      </div>
    </div>
  );

  const contentArea = (
    <>
      <MarkdownContent content={message.content} />
      {message.attachments?.map((att) => (
        <AttachmentView key={att.id} attachment={att} />
      ))}
    </>
  );

  const ctxMenu = contextMenu && (
    <MessageContextMenu
      message={message}
      x={contextMenu.x}
      y={contextMenu.y}
      onClose={() => setContextMenu(null)}
      onEdit={() => {
        setEditing(true);
        setEditContent(message.content);
      }}
    />
  );

  if (grouped) {
    return (
      <div
        className="group flex gap-4 px-4 py-[2px] hover:bg-white/[0.02]"
        onContextMenu={handleContextMenu}
      >
        <div className="flex w-10 shrink-0 items-center justify-center">
          <span className="hidden text-[10px] text-text-muted group-hover:inline">
            {new Date(message.created_at).toLocaleTimeString([], {
              hour: "numeric",
              minute: "2-digit",
            })}
          </span>
        </div>
        <div className="min-w-0 flex-1">
          {editing ? editArea : contentArea}
        </div>
        {ctxMenu}
      </div>
    );
  }

  const profileCard = profilePopover && member && selectedGuildId && (
    <UserProfilePopover
      member={member}
      displayName={displayName}
      username={message.author_username}
      avatarHash={message.author_avatar_hash}
      guildId={selectedGuildId}
      x={profilePopover.x}
      y={profilePopover.y}
      onClose={() => setProfilePopover(null)}
    />
  );

  return (
    <div
      className="group mt-[17px] flex gap-4 px-4 py-0.5 hover:bg-white/[0.02]"
      onContextMenu={handleContextMenu}
    >
      <div className="mt-0.5">
        <AvatarView
          userId={message.author_id}
          displayName={displayName}
          avatarHash={message.author_avatar_hash}
          size="lg"
        />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-2">
          <button
            onClick={handleNameClick}
            className="font-medium text-text-primary hover:underline"
          >
            {displayName}
          </button>
          <span className="text-xs text-text-muted">
            {formatTimestamp(message.created_at)}
          </span>
          {message.edited_at && (
            <span className="text-xs text-text-muted">(edited)</span>
          )}
        </div>
        {editing ? editArea : contentArea}
      </div>
      {ctxMenu}
      {profileCard}
    </div>
  );
}

export default function MessageList({
  channelId,
}: {
  channelId: string;
}) {
  const messages = useMessagesStore(
    (s) => s.messagesByChannel.get(channelId) || [],
  );
  const hasMore = useMessagesStore(
    (s) => s.hasMore.get(channelId) ?? true,
  );
  const setMessages = useMessagesStore((s) => s.setMessages);
  const prependMessages = useMessagesStore((s) => s.prependMessages);
  const fetchMessages = useMessagesStore((s) => s.fetchMessages);

  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [showNewIndicator, setShowNewIndicator] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const prevScrollHeightRef = useRef(0);
  const isAtBottomRef = useRef(true);
  const prevMessageCountRef = useRef(0);

  const updateIsAtBottom = useCallback(() => {
    if (!listRef.current) return;
    const el = listRef.current;
    isAtBottomRef.current =
      el.scrollHeight - el.scrollTop - el.clientHeight < 50;
  }, []);

  // Initial load
  useEffect(() => {
    setLoading(true);
    prevMessageCountRef.current = 0;
    setShowNewIndicator(false);
    fetchMessages(channelId)
      .then((msgs) => {
        setMessages(channelId, [...msgs].reverse());
        requestAnimationFrame(() => {
          bottomRef.current?.scrollIntoView();
          isAtBottomRef.current = true;
        });
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [channelId, fetchMessages, setMessages]);

  // Handle new messages
  useEffect(() => {
    if (messages.length <= prevMessageCountRef.current) {
      prevMessageCountRef.current = messages.length;
      return;
    }
    prevMessageCountRef.current = messages.length;

    if (isAtBottomRef.current) {
      requestAnimationFrame(() => {
        bottomRef.current?.scrollIntoView({ behavior: "smooth" });
      });
    } else {
      setShowNewIndicator(true);
    }
  }, [messages.length]);

  function scrollToBottom() {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    setShowNewIndicator(false);
  }

  const loadOlder = useCallback(async () => {
    if (loadingMore || !hasMore || messages.length === 0) return;
    setLoadingMore(true);
    const oldestId = messages[0]?.id;
    if (!oldestId) {
      setLoadingMore(false);
      return;
    }
    prevScrollHeightRef.current = listRef.current?.scrollHeight || 0;
    try {
      const older = await fetchMessages(channelId, oldestId);
      if (older.length > 0) {
        prependMessages(channelId, [...older].reverse());
        requestAnimationFrame(() => {
          if (listRef.current) {
            const newHeight = listRef.current.scrollHeight;
            listRef.current.scrollTop =
              newHeight - prevScrollHeightRef.current;
          }
        });
      }
    } catch {
      // ignore
    } finally {
      setLoadingMore(false);
    }
  }, [
    channelId,
    loadingMore,
    hasMore,
    messages,
    fetchMessages,
    prependMessages,
  ]);

  const handleScroll = useCallback(() => {
    updateIsAtBottom();
    if (isAtBottomRef.current) {
      setShowNewIndicator(false);
    }
    if (!listRef.current) return;
    if (listRef.current.scrollTop < 100) {
      loadOlder();
    }
  }, [loadOlder, updateIsAtBottom]);

  if (loading) {
    return (
      <div className="flex flex-1 items-center justify-center text-text-muted">
        Loading messages...
      </div>
    );
  }

  if (messages.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center text-text-muted">
        No messages yet. Say something!
      </div>
    );
  }

  // Build elements with date separators and author grouping
  const elements: React.ReactNode[] = [];
  for (let i = 0; i < messages.length; i++) {
    const msg = messages[i];
    const prev = i > 0 ? messages[i - 1] : null;

    if (!prev || isDifferentDay(prev.created_at, msg.created_at)) {
      elements.push(
        <DateSeparator key={`date-${msg.id}`} date={msg.created_at} />,
      );
    }

    const grouped = prev !== null && shouldGroup(prev, msg);
    elements.push(
      <MessageItem key={msg.id} message={msg} grouped={grouped} />,
    );
  }

  return (
    <div className="relative flex flex-1 flex-col overflow-hidden">
      <div
        ref={listRef}
        onScroll={handleScroll}
        className="flex flex-1 flex-col overflow-y-auto pb-2"
      >
        {loadingMore && (
          <div className="py-2 text-center text-sm text-text-muted">
            Loading older messages...
          </div>
        )}
        {!hasMore && messages.length > 0 && (
          <div className="py-4 text-center text-sm text-text-muted">
            Beginning of conversation
          </div>
        )}
        {elements}
        <div ref={bottomRef} />
      </div>

      {showNewIndicator && (
        <button
          onClick={scrollToBottom}
          className="absolute bottom-2 left-1/2 -translate-x-1/2 rounded-full bg-accent px-4 py-1.5 text-sm font-medium text-white shadow-lg hover:bg-accent-hover"
        >
          New messages below
        </button>
      )}
    </div>
  );
}
