package database

// Repository handles all database operations
// Methods are organized across multiple files by domain:
// - bot.go: Bot CRUD operations
// - schedule.go: Scheduled messages operations
// - auto_reply.go: Auto-reply and custom commands
// - user.go: Message logs, user analytics, and bans
// - forced_sub.go: Forced channel subscription operations
type Repository struct {
	mysql         *MySQL
	encryptionKey string
}

// NewRepository creates a new repository instance
func NewRepository(mysql *MySQL, encryptionKey string) *Repository {
	return &Repository{
		mysql:         mysql,
		encryptionKey: encryptionKey,
	}
}
