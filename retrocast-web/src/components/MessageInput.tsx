import { useState, useRef, useEffect, useCallback, type KeyboardEvent } from "react";
import { api } from "@/lib/api";
import type { Message, Attachment } from "@/types";

const TYPING_THROTTLE = 8000;

interface MessageInputProps {
  channelId: string;
  channelName?: string;
  droppedFiles?: File[];
  onDropConsumed?: () => void;
}

export default function MessageInput({
  channelId,
  channelName,
  droppedFiles,
  onDropConsumed,
}: MessageInputProps) {
  const [content, setContent] = useState("");
  const [sending, setSending] = useState(false);
  const [pendingFiles, setPendingFiles] = useState<File[]>([]);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const lastTypingRef = useRef(0);

  const sendTyping = useCallback(() => {
    const now = Date.now();
    if (now - lastTypingRef.current < TYPING_THROTTLE) return;
    lastTypingRef.current = now;
    api.post(`/api/v1/channels/${channelId}/typing`).catch(() => {});
  }, [channelId]);

  function addFiles(files: File[]) {
    setPendingFiles((prev) => [...prev, ...files]);
  }

  // Consume files dropped via FileUpload wrapper
  useEffect(() => {
    if (droppedFiles && droppedFiles.length > 0) {
      addFiles(droppedFiles);
      onDropConsumed?.();
    }
  }, [droppedFiles]);

  function handleInput(value: string) {
    setContent(value);
    if (value.length > 0) {
      sendTyping();
    }
  }

  async function handleSend() {
    const text = content.trim();
    if (!text && pendingFiles.length === 0) return;

    setSending(true);
    try {
      // Upload pending files first
      for (const file of pendingFiles) {
        await api.upload<Attachment>(
          `/api/v1/channels/${channelId}/attachments`,
          file,
        );
      }

      // Send message
      if (text) {
        await api.post<Message>(`/api/v1/channels/${channelId}/messages`, {
          content: text,
        });
      }

      setContent("");
      setPendingFiles([]);
      lastTypingRef.current = 0;
    } catch {
      // silent
    } finally {
      setSending(false);
    }
  }

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  function handleFileSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const files = e.target.files;
    if (files) {
      addFiles(Array.from(files));
    }
    e.target.value = "";
  }

  function removePendingFile(index: number) {
    setPendingFiles((prev) => prev.filter((_, i) => i !== index));
  }

  const placeholder = channelName
    ? `Message #${channelName}`
    : "Message this channel";

  return (
    <div className="px-4 pb-6">
      {/* Pending files with image previews */}
      {pendingFiles.length > 0 && (
        <div className="mb-2 flex flex-wrap gap-2">
          {pendingFiles.map((file, i) => {
            const isImage = file.type.startsWith("image/");
            return (
              <div
                key={`${file.name}-${i}`}
                className="relative flex items-center gap-1 rounded bg-bg-secondary p-1.5 text-sm text-text-secondary"
              >
                {isImage && (
                  <img
                    src={URL.createObjectURL(file)}
                    alt={file.name}
                    className="h-16 w-16 rounded object-cover"
                  />
                )}
                {!isImage && (
                  <span className="max-w-32 truncate px-1">{file.name}</span>
                )}
                <button
                  onClick={() => removePendingFile(i)}
                  className="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-xs text-white hover:bg-red-600"
                >
                  x
                </button>
              </div>
            );
          })}
        </div>
      )}

      <div className="flex items-end gap-2 rounded-lg bg-bg-input px-4 py-2">
        <button
          onClick={() => fileInputRef.current?.click()}
          className="mb-0.5 shrink-0 text-text-muted hover:text-text-primary"
          title="Upload a file"
        >
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="currentColor"
          >
            <path d="M12 2a1 1 0 0 1 1 1v8h8a1 1 0 1 1 0 2h-8v8a1 1 0 1 1-2 0v-8H3a1 1 0 1 1 0-2h8V3a1 1 0 0 1 1-1Z" />
          </svg>
        </button>
        <input
          ref={fileInputRef}
          type="file"
          className="hidden"
          onChange={handleFileSelect}
          multiple
        />

        <textarea
          value={content}
          onChange={(e) => handleInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={sending}
          rows={1}
          className="max-h-48 min-h-[24px] flex-1 resize-none bg-transparent text-sm text-text-primary placeholder:text-text-muted focus:outline-none"
          style={{
            height: "auto",
            overflow: "hidden",
          }}
          onInput={(e) => {
            const el = e.currentTarget;
            el.style.height = "auto";
            el.style.height = `${Math.min(el.scrollHeight, 192)}px`;
            el.style.overflow =
              el.scrollHeight > 192 ? "auto" : "hidden";
          }}
        />
      </div>
    </div>
  );
}
