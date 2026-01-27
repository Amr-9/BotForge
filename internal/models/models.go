package models

import "time"

// Bot represents a child bot registered by a user
type Bot struct {
	ID           int64     `db:"id"`
	Token        string    `db:"token"`
	OwnerChatID  int64     `db:"owner_chat_id"`
	IsActive     bool      `db:"is_active"`
	StartMessage string    `db:"start_message"`
	CreatedAt    time.Time `db:"created_at"`
}

// MessageLog stores the mapping between admin message and user chat
type MessageLog struct {
	ID         int64     `db:"id"`
	AdminMsgID int       `db:"admin_msg_id"`
	UserChatID int64     `db:"user_chat_id"`
	BotID      int64     `db:"bot_id"`
	CreatedAt  time.Time `db:"created_at"`
}

// BannedUser represents a banned user for a specific bot
type BannedUser struct {
	ID         int64     `db:"id"`
	BotID      int64     `db:"bot_id"`
	UserChatID int64     `db:"user_chat_id"`
	BannedBy   int64     `db:"banned_by"`
	CreatedAt  time.Time `db:"created_at"`
}
