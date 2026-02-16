export interface User {
  id: string;
  username: string;
  display_name: string;
  avatar_hash: string | null;
  created_at: string;
}

export interface Guild {
  id: string;
  name: string;
  icon_hash: string | null;
  owner_id: string;
  created_at: string;
}

export interface Channel {
  id: string;
  guild_id: string;
  name: string;
  type: number;
  position: number;
  topic: string | null;
  parent_id: string | null;
}

export interface Message {
  id: string;
  channel_id: string;
  author_id: string;
  content: string;
  created_at: string;
  edited_at: string | null;
  author_username: string;
  author_display_name: string;
  author_avatar_hash: string | null;
  attachments: Attachment[];
}

export interface Attachment {
  id: string;
  message_id: string;
  filename: string;
  content_type: string;
  size: number;
  url: string;
}

export interface Role {
  id: string;
  guild_id: string;
  name: string;
  color: number;
  permissions: string;
  position: number;
  is_default: boolean;
}

export interface Member {
  guild_id: string;
  user_id: string;
  nickname: string | null;
  joined_at: string;
  roles: string[];
}

export interface DMChannel {
  id: string;
  type: number;
  recipients: User[];
  created_at: string;
}

export interface Invite {
  code: string;
  guild_id: string;
  channel_id: string | null;
  creator_id: string;
  max_uses: number;
  uses: number;
  expires_at: string | null;
  created_at: string;
}

export interface GatewayPayload {
  op: number;
  d: unknown;
  s?: number;
  t?: string;
}
