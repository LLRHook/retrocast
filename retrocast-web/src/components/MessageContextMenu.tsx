import { useEffect } from "react";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/lib/api";
import type { Message } from "@/types";

interface MessageContextMenuProps {
  message: Message;
  x: number;
  y: number;
  onClose: () => void;
  onEdit: () => void;
}

export default function MessageContextMenu({
  message,
  x,
  y,
  onClose,
  onEdit,
}: MessageContextMenuProps) {
  const currentUser = useAuthStore((s) => s.user);
  const isOwn = currentUser?.id === message.author_id;

  useEffect(() => {
    function handler() {
      onClose();
    }
    document.addEventListener("click", handler);
    return () => document.removeEventListener("click", handler);
  }, [onClose]);

  async function handleDelete() {
    if (!confirm("Delete this message?")) {
      onClose();
      return;
    }
    try {
      await api.delete(
        `/api/v1/channels/${message.channel_id}/messages/${message.id}`,
      );
    } catch {
      // ignore
    }
    onClose();
  }

  function handleCopy() {
    navigator.clipboard.writeText(message.content).catch(() => {});
    onClose();
  }

  function handleCopyId() {
    navigator.clipboard.writeText(message.id).catch(() => {});
    onClose();
  }

  function handleEdit() {
    onEdit();
    onClose();
  }

  // Ensure menu stays within viewport
  const menuStyle: React.CSSProperties = {
    left: Math.min(x, window.innerWidth - 200),
    top: Math.min(y, window.innerHeight - 200),
  };

  return (
    <div
      className="fixed z-50 min-w-44 rounded-md bg-bg-tertiary py-1.5 shadow-lg"
      style={menuStyle}
    >
      <button
        onClick={handleCopy}
        className="flex w-full items-center px-3 py-1.5 text-left text-sm text-text-secondary hover:bg-accent hover:text-white"
      >
        Copy Text
      </button>
      <button
        onClick={handleCopyId}
        className="flex w-full items-center px-3 py-1.5 text-left text-sm text-text-secondary hover:bg-accent hover:text-white"
      >
        Copy Message ID
      </button>
      {isOwn && (
        <>
          <div className="my-1.5 border-t border-border" />
          <button
            onClick={handleEdit}
            className="flex w-full items-center px-3 py-1.5 text-left text-sm text-text-secondary hover:bg-accent hover:text-white"
          >
            Edit Message
          </button>
          <button
            onClick={handleDelete}
            className="flex w-full items-center px-3 py-1.5 text-left text-sm text-red-400 hover:bg-red-500 hover:text-white"
          >
            Delete Message
          </button>
        </>
      )}
    </div>
  );
}
