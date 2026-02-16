import PresenceDot from "@/components/PresenceDot";

interface AvatarViewProps {
  userId: string;
  displayName: string;
  avatarHash: string | null;
  size?: "sm" | "md" | "lg" | "xl";
  showPresence?: boolean;
  className?: string;
}

const SIZE_MAP = {
  sm: { container: "h-6 w-6", text: "text-[10px]", presence: "sm" as const, px: 24 },
  md: { container: "h-8 w-8", text: "text-xs", presence: "sm" as const, px: 32 },
  lg: { container: "h-10 w-10", text: "text-sm", presence: "md" as const, px: 40 },
  xl: { container: "h-20 w-20", text: "text-2xl", presence: "lg" as const, px: 80 },
};

// 8 preset avatar background colors derived from user ID
const AVATAR_COLORS = [
  "bg-red-500",
  "bg-orange-500",
  "bg-amber-500",
  "bg-emerald-500",
  "bg-cyan-500",
  "bg-blue-500",
  "bg-violet-500",
  "bg-pink-500",
];

function avatarColor(userId: string): string {
  let hash = 0;
  for (let i = 0; i < userId.length; i++) {
    hash = (hash * 31 + userId.charCodeAt(i)) | 0;
  }
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}

export default function AvatarView({
  userId,
  displayName,
  avatarHash,
  size = "md",
  showPresence = false,
  className = "",
}: AvatarViewProps) {
  const serverUrl = localStorage.getItem("serverUrl") || "";
  const initial = displayName[0]?.toUpperCase() || "?";
  const s = SIZE_MAP[size];
  const bgColor = avatarColor(userId);

  return (
    <div className={`relative shrink-0 ${className}`}>
      <div
        className={`flex items-center justify-center rounded-full ${bgColor} ${s.container}`}
      >
        {avatarHash ? (
          <img
            src={`${serverUrl}/api/v1/users/${userId}/avatar`}
            alt={displayName}
            className={`rounded-full object-cover ${s.container}`}
            loading="lazy"
          />
        ) : (
          <span className={`font-medium text-white ${s.text}`}>{initial}</span>
        )}
      </div>
      {showPresence && (
        <div className="absolute -bottom-0.5 -right-0.5">
          <PresenceDot userId={userId} size={s.presence} />
        </div>
      )}
    </div>
  );
}
