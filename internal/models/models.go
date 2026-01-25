package models

import "time"

// Bot represents a child bot registered by a user
type Bot struct {
	ID          int64     `db:"id"`
	Token       string    `db:"token"`
	OwnerChatID int64     `db:"owner_chat_id"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
}

// MessageLog stores the mapping between admin message and user chat
type MessageLog struct {
	ID         int64     `db:"id"`
	AdminMsgID int       `db:"admin_msg_id"`
	UserChatID int64     `db:"user_chat_id"`
	BotToken   string    `db:"bot_token"`
	CreatedAt  time.Time `db:"created_at"`
}
