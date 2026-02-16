package database

import (
	"context"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id int64) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int64) error
}

type GuildRepository interface {
	Create(ctx context.Context, guild *models.Guild) error
	GetByID(ctx context.Context, id int64) (*models.Guild, error)
	Update(ctx context.Context, guild *models.Guild) error
	Delete(ctx context.Context, id int64) error
	GetByUserID(ctx context.Context, userID int64) ([]models.Guild, error)
}

type ChannelRepository interface {
	Create(ctx context.Context, channel *models.Channel) error
	GetByID(ctx context.Context, id int64) (*models.Channel, error)
	GetByGuildID(ctx context.Context, guildID int64) ([]models.Channel, error)
	Update(ctx context.Context, channel *models.Channel) error
	Delete(ctx context.Context, id int64) error
}

type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) error
	GetByID(ctx context.Context, id int64) (*models.Role, error)
	GetByGuildID(ctx context.Context, guildID int64) ([]models.Role, error)
	Update(ctx context.Context, role *models.Role) error
	Delete(ctx context.Context, id int64) error
	GetByMember(ctx context.Context, guildID, userID int64) ([]models.Role, error)
}

type MemberRepository interface {
	Create(ctx context.Context, member *models.Member) error
	GetByGuildAndUser(ctx context.Context, guildID, userID int64) (*models.Member, error)
	GetByGuildID(ctx context.Context, guildID int64, limit, offset int) ([]models.Member, error)
	Update(ctx context.Context, member *models.Member) error
	Delete(ctx context.Context, guildID, userID int64) error
	AddRole(ctx context.Context, guildID, userID, roleID int64) error
	RemoveRole(ctx context.Context, guildID, userID, roleID int64) error
}

type MessageRepository interface {
	Create(ctx context.Context, msg *models.Message) error
	GetByID(ctx context.Context, id int64) (*models.MessageWithAuthor, error)
	GetByChannelID(ctx context.Context, channelID int64, before *int64, limit int) ([]models.MessageWithAuthor, error)
	Update(ctx context.Context, msg *models.Message) error
	Delete(ctx context.Context, id int64) error
	SearchMessages(ctx context.Context, guildID int64, query string, authorID *int64, before *time.Time, after *time.Time, limit int) ([]models.MessageWithAuthor, error)
}

type InviteRepository interface {
	Create(ctx context.Context, invite *models.Invite) error
	GetByCode(ctx context.Context, code string) (*models.Invite, error)
	GetByGuildID(ctx context.Context, guildID int64) ([]models.Invite, error)
	IncrementUses(ctx context.Context, code string) error
	Delete(ctx context.Context, code string) error
}

type ChannelOverrideRepository interface {
	Set(ctx context.Context, override *models.ChannelOverride) error
	GetByChannel(ctx context.Context, channelID int64) ([]models.ChannelOverride, error)
	Delete(ctx context.Context, channelID, roleID int64) error
}

type AttachmentRepository interface {
	Create(ctx context.Context, attachment *models.Attachment) error
	GetByMessageID(ctx context.Context, messageID int64) ([]models.Attachment, error)
	Delete(ctx context.Context, id int64) error
}

type BanRepository interface {
	Create(ctx context.Context, ban *models.Ban) error
	GetByGuildAndUser(ctx context.Context, guildID, userID int64) (*models.Ban, error)
	GetByGuildID(ctx context.Context, guildID int64) ([]models.Ban, error)
	Delete(ctx context.Context, guildID, userID int64) error
}

type DMChannelRepository interface {
	Create(ctx context.Context, dm *models.DMChannel) error
	GetByID(ctx context.Context, id int64) (*models.DMChannel, error)
	GetByUserID(ctx context.Context, userID int64) ([]models.DMChannel, error)
	GetOrCreateDM(ctx context.Context, user1ID, user2ID, newID int64) (*models.DMChannel, error)
	AddRecipient(ctx context.Context, channelID, userID int64) error
	IsRecipient(ctx context.Context, channelID, userID int64) (bool, error)
}

type ReadStateRepository interface {
	Upsert(ctx context.Context, userID, channelID, lastMessageID int64) error
	GetByUser(ctx context.Context, userID int64) ([]models.ReadState, error)
	GetByUserAndChannel(ctx context.Context, userID, channelID int64) (*models.ReadState, error)
	IncrementMentionCount(ctx context.Context, userID, channelID int64) error
}

type ReactionRepository interface {
	Add(ctx context.Context, messageID, userID int64, emoji string) error
	Remove(ctx context.Context, messageID, userID int64, emoji string) error
	GetByMessage(ctx context.Context, messageID int64) ([]models.Reaction, error)
	GetCountsByMessage(ctx context.Context, messageID, currentUserID int64) ([]models.ReactionCount, error)
	GetUsersByReaction(ctx context.Context, messageID int64, emoji string, limit int) ([]int64, error)
}
