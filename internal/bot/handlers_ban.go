package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"gopkg.in/telebot.v3"
)

// handleBanCommand processes the "ban" or "/ban" command when admin replies to a user message
func (m *Manager) handleBanCommand(ctx context.Context, c telebot.Context, bot *telebot.Bot, token string, userChatID int64) error {
	m.mu.RLock()
	botID := m.botIDs[token]
	m.mu.RUnlock()

	// Check if user is already banned
	isBanned, err := m.repo.IsUserBanned(ctx, botID, userChatID)
	if err != nil {
		log.Printf("Error checking ban status: %v", err)
		return c.Reply("Failed to check user status.")
	}

	if isBanned {
		return c.Reply("This user is already banned.")
	}

	// Ban the user
	if err := m.repo.BanUser(ctx, botID, userChatID, c.Sender().ID); err != nil {
		log.Printf("Error banning user: %v", err)
		return c.Reply("Failed to ban user.")
	}

	// Update cache
	m.cache.SetUserBanned(ctx, token, userChatID)
	m.cache.InvalidateNotBannedCache(ctx, token, userChatID)

	// Send ban notification to the user (one-time message)
	userChat := &telebot.Chat{ID: userChatID}
	bot.Send(userChat, "You have been blocked from sending messages to this bot.")

	// Get user info for confirmation to admin
	chat, err := bot.ChatByID(userChatID)
	userName := fmt.Sprintf("<code>%d</code>", userChatID)
	if err == nil && chat != nil {
		userName = formatBanUserName(chat)
	}

	return c.Reply(fmt.Sprintf("🚫 <b>User Banned</b>\n\n%s has been banned from this bot.", userName), telebot.ModeHTML)
}

// formatBanUserName creates a display name from chat info
func formatBanUserName(chat *telebot.Chat) string {
	name := chat.FirstName
	if chat.LastName != "" {
		name += " " + chat.LastName
	}
	if chat.Username != "" {
		name += " (@" + chat.Username + ")"
	}
	return name
}

// handleBannedUsersList shows the list of banned users with unban buttons
func (m *Manager) handleBannedUsersList(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Get page from callback data (default: 0)
		page := 0
		if c.Callback() != nil && c.Callback().Data != "" {
			// Check if it's just a page number (not an unban action)
			if p, err := strconv.Atoi(c.Callback().Data); err == nil {
				page = p
			}
		}

		pageSize := 5
		offset := page * pageSize

		// Get banned users count
		count, err := m.repo.GetBannedUserCount(ctx, botID)
		if err != nil {
			return c.Edit("Failed to retrieve banned users.")
		}

		if count == 0 {
			menu := &telebot.ReplyMarkup{}
			btnBack := menu.Data("« Back to Settings", "child_settings")
			menu.Inline(menu.Row(btnBack))
			return c.Edit("🚫 <b>Banned Users</b>\n\nNo users are currently banned.", menu, telebot.ModeHTML)
		}

		// Get banned users
		bannedUsers, err := m.repo.GetBannedUsers(ctx, botID, pageSize, offset)
		if err != nil {
			return c.Edit("Failed to retrieve banned users.")
		}

		// Build message
		msg := fmt.Sprintf("🚫 <b>Banned Users</b> (%d total)\n\n", count)

		menu := &telebot.ReplyMarkup{}
		var rows []telebot.Row

		for i, ban := range bannedUsers {
			chat, _ := bot.ChatByID(ban.UserChatID)
			name := fmt.Sprintf("%d", ban.UserChatID)
			if chat != nil {
				if chat.FirstName != "" {
					name = chat.FirstName
					if chat.LastName != "" {
						name += " " + chat.LastName
					}
				}
				if chat.Username != "" {
					name += " (@" + chat.Username + ")"
				}
			}
			msg += fmt.Sprintf("%d. %s\n   🆔 <code>%d</code>\n   📅 %s\n\n",
				offset+i+1, name, ban.UserChatID, ban.CreatedAt.Format("2006-01-02 15:04"))

			// Add unban button for each user
			btnUnban := menu.Data(fmt.Sprintf("Unban %d", ban.UserChatID), "unban_user", strconv.FormatInt(ban.UserChatID, 10))
			rows = append(rows, menu.Row(btnUnban))
		}

		// Pagination buttons
		var navRow []telebot.Btn
		if page > 0 {
			navRow = append(navRow, menu.Data("« Prev", "banned_list", strconv.Itoa(page-1)))
		}
		if int64(offset+pageSize) < count {
			navRow = append(navRow, menu.Data("Next »", "banned_list", strconv.Itoa(page+1)))
		}
		if len(navRow) > 0 {
			rows = append(rows, menu.Row(navRow...))
		}

		btnBack := menu.Data("« Back to Settings", "child_settings")
		rows = append(rows, menu.Row(btnBack))

		menu.Inline(rows...)

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleUnbanUser processes the unban button click from banned users list
func (m *Manager) handleUnbanUser(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Get user ID from callback data
		data := c.Callback().Data
		userChatID, err := strconv.ParseInt(data, 10, 64)
		if err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Invalid user ID", ShowAlert: true})
		}

		// Unban the user
		if err := m.repo.UnbanUser(ctx, botID, userChatID); err != nil {
			log.Printf("Error unbanning user: %v", err)
			return c.Respond(&telebot.CallbackResponse{Text: "Failed to unban user", ShowAlert: true})
		}

		// Update cache
		m.cache.RemoveUserBan(ctx, token, userChatID)

		// Show success message
		c.Respond(&telebot.CallbackResponse{Text: "User unbanned successfully!", ShowAlert: false})

		// Refresh the banned users list
		return m.handleBannedUsersList(bot, token, ownerChat)(c)
	}
}

// checkUserBanned checks if a user is banned with cache-through pattern
func (m *Manager) checkUserBanned(ctx context.Context, token string, botID, userChatID int64) (bool, error) {
	// Check positive cache first (user is banned)
	isBanned, cacheHit, err := m.cache.IsUserBanned(ctx, token, userChatID)
	if err != nil {
		log.Printf("Cache error checking ban: %v", err)
	}
	if cacheHit && isBanned {
		return true, nil
	}

	// Check negative cache (user is not banned)
	notBannedCached, err := m.cache.IsNotBannedCached(ctx, token, userChatID)
	if err != nil {
		log.Printf("Cache error checking not-banned: %v", err)
	}
	if notBannedCached {
		return false, nil
	}

	// Check database
	isBanned, err = m.repo.IsUserBanned(ctx, botID, userChatID)
	if err != nil {
		return false, err
	}

	// Update cache
	if isBanned {
		m.cache.SetUserBanned(ctx, token, userChatID)
	} else {
		m.cache.CacheNotBanned(ctx, token, userChatID)
	}

	return isBanned, nil
}
