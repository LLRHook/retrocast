import { useTypingStore } from "@/stores/typing";
import { useAuthStore } from "@/stores/auth";
import { useMembersStore } from "@/stores/members";
import { useGuildsStore } from "@/stores/guilds";

export default function TypingIndicator({
  channelId,
}: {
  channelId: string;
}) {
  const typingMap = useTypingStore(
    (s) => s.typingByChannel.get(channelId),
  );
  const currentUserId = useAuthStore((s) => s.user?.id);
  const selectedGuildId = useGuildsStore((s) => s.selectedGuildId);
  const membersByGuild = useMembersStore((s) => s.membersByGuild);

  if (!typingMap || typingMap.size === 0) {
    return <div className="h-6" />;
  }

  // Filter out current user
  const typingUserIds = Array.from(typingMap.keys()).filter(
    (id) => id !== currentUserId,
  );

  if (typingUserIds.length === 0) {
    return <div className="h-6" />;
  }

  // Resolve display names
  const guildMembers = selectedGuildId
    ? membersByGuild.get(selectedGuildId)
    : null;

  const names = typingUserIds.map((userId) => {
    const member = guildMembers?.get(userId);
    return member?.nickname || userId;
  });

  let text: string;
  if (names.length === 1) {
    text = `${names[0]} is typing...`;
  } else if (names.length === 2) {
    text = `${names[0]} and ${names[1]} are typing...`;
  } else if (names.length === 3) {
    text = `${names[0]}, ${names[1]}, and ${names[2]} are typing...`;
  } else {
    text = "Several people are typing...";
  }

  return (
    <div className="flex h-6 items-center px-4 text-xs text-text-muted">
      <span className="mr-1 inline-flex gap-0.5">
        <span className="inline-block h-1 w-1 animate-bounce rounded-full bg-text-muted" />
        <span className="inline-block h-1 w-1 animate-bounce rounded-full bg-text-muted [animation-delay:0.15s]" />
        <span className="inline-block h-1 w-1 animate-bounce rounded-full bg-text-muted [animation-delay:0.3s]" />
      </span>
      {text}
    </div>
  );
}
