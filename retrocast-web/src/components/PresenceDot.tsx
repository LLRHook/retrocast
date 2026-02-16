import { usePresenceStore } from "@/stores/presence";

const STATUS_COLORS: Record<string, string> = {
  online: "bg-[#23a559]",
  idle: "bg-[#f0b232]",
  dnd: "bg-[#f23f43]",
  invisible: "bg-[#80848e]",
  offline: "bg-[#80848e]",
};

interface PresenceDotProps {
  userId: string;
  size?: "sm" | "md" | "lg";
  className?: string;
}

export default function PresenceDot({
  userId,
  size = "md",
  className = "",
}: PresenceDotProps) {
  const status = usePresenceStore((s) => s.presences.get(userId) || "offline");

  const sizeClasses = {
    sm: "h-[10px] w-[10px]",
    md: "h-3 w-3",
    lg: "h-4 w-4",
  };

  return (
    <span
      className={`inline-block rounded-full border-2 border-bg-secondary ${STATUS_COLORS[status]} ${sizeClasses[size]} ${className}`}
      title={status}
    />
  );
}
