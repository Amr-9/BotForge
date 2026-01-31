package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

// getBotUsername retrieves the bot's username from Telegram API
func getBotUsername(token string) string {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", token)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to get bot username: %v", err)
		return "Unknown"
	}
	defer resp.Body.Close()

	var result struct {
		Ok     bool `json:"ok"`
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode bot info: %v", err)
		return "Unknown"
	}

	if !result.Ok || result.Result.Username == "" {
		return "Unknown"
	}

	return result.Result.Username
}

// handleAddBotBtn handles add bot button
func (f *Factory) handleAddBotBtn(c telebot.Context) error {
	msg := `📝 <b>Add New Bot</b>

Please send me your bot token.

<b>How to get a token:</b>
1. Open @BotFather
2. Send /newbot and follow instructions
3. Copy the token and send it here

<i>Your token looks like:</i>
<code>123456789:ABCdefGHIjklMNOpqrsTUVwxyz</code>`

	return c.Edit(msg, f.getBackButton(), telebot.ModeHTML)
}

// handleMyBotsBtn lists all bots owned by the user
func (f *Factory) handleMyBotsBtn(c telebot.Context) error {
	ctx := context.Background()
	senderID := c.Sender().ID

	bots, err := f.repo.GetBotsByOwner(ctx, senderID)
	if err != nil {
		log.Printf("Failed to get bots: %v", err)
		return c.Edit("❌ Failed to retrieve your bots. Please try again.", f.getBackButton())
	}

	if len(bots) == 0 {
		msg := `📭 <b>No Bots Found</b>

You don't have any bots yet.
Use "Add Bot" to add your first bot!`
		return c.Edit(msg, f.getBackButton(), telebot.ModeHTML)
	}

	msg := fmt.Sprintf("🤖 <b>Your Bots (%d)</b>\n\n", len(bots))
	msg += "Select a bot to manage:\n"

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	for _, bot := range bots {
		status := "🔴"
		if f.manager.IsRunning(bot.Token) {
			status = "🟢"
		}

		// Get bot username - use stored value or fetch from API
		username := bot.Username
		if username == "" {
			// No stored username, fetch from Telegram API
			username = getBotUsername(bot.Token)
			if username != "" && username != "Unknown" {
				// Save to database for future use
				if err := f.repo.UpdateBotUsername(ctx, bot.ID, username); err != nil {
					log.Printf("Failed to save bot username to DB: %v", err)
				}
			}
		}

		btnText := fmt.Sprintf("%s @%s", status, username)

		btn := menu.Data(btnText, CallbackBotSelect, bot.Token[:20])
		rows = append(rows, menu.Row(btn))
	}

	// Add back button
	btnBack := menu.Data("« Back to Menu", CallbackMainMenu)
	rows = append(rows, menu.Row(btnBack))

	menu.Inline(rows...)

	return c.Edit(msg, menu, telebot.ModeHTML)
}

// handleBotDetails shows details for a specific bot
func (f *Factory) handleBotDetails(c telebot.Context, tokenPrefix string) error {
	ctx := context.Background()
	senderID := c.Sender().ID

	// Find the full token
	bots, err := f.repo.GetBotsByOwner(ctx, senderID)
	if err != nil {
		return c.Edit("❌ Error loading bot.", f.getBackButton())
	}

	var targetBot *struct {
		id          int64
		token       string
		username    string
		ownerChatID int64
		createdAt   time.Time
	}

	for _, bot := range bots {
		if strings.HasPrefix(bot.Token, tokenPrefix) {
			targetBot = &struct {
				id          int64
				token       string
				username    string
				ownerChatID int64
				createdAt   time.Time
			}{id: bot.ID, token: bot.Token, username: bot.Username, ownerChatID: bot.OwnerChatID, createdAt: bot.CreatedAt}
			break
		}
	}

	if targetBot == nil {
		return c.Edit("❌ Bot not found.", f.getBackButton())
	}

	isRunning := f.manager.IsRunning(targetBot.token)
	status := "🔴 Stopped"
	if isRunning {
		status = "🟢 Running"
	}

	// Get bot username - use stored value or fetch from API
	username := targetBot.username
	if username == "" {
		// No stored username, fetch from Telegram API
		username = getBotUsername(targetBot.token)
		if username != "" && username != "Unknown" {
			// Save to database for future use
			if err := f.repo.UpdateBotUsername(ctx, targetBot.id, username); err != nil {
				log.Printf("Failed to save bot username to DB: %v", err)
			}
		}
	}

	// Format created date
	createdAt := targetBot.createdAt.Format("2006-01-02 3:04 PM")

	msg := fmt.Sprintf(`🤖 <b>Bot Details</b>

<b>Username:</b> @%s
<b>Token:</b> <code>%s</code>
<b>Status:</b> %s
<b>Created At:</b> %s

Select an action:`, username, maskToken(targetBot.token), status, createdAt)

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	if isRunning {
		btnStop := menu.Data("⏹ Stop Bot", CallbackStopBot, tokenPrefix)
		rows = append(rows, menu.Row(btnStop))
	} else {
		btnStart := menu.Data("▶️ Start Bot", CallbackStartBot, tokenPrefix)
		rows = append(rows, menu.Row(btnStart))
	}

	btnDelete := menu.Data("🗑 Delete Bot", CallbackDeleteBot, tokenPrefix)
	btnBack := menu.Data("« Back to Bots", CallbackMyBots)

	rows = append(rows, menu.Row(btnDelete))
	rows = append(rows, menu.Row(btnBack))

	menu.Inline(rows...)

	return c.Edit(msg, menu, telebot.ModeHTML)
}

// handleStartBotAction starts a bot
func (f *Factory) handleStartBotAction(c telebot.Context, tokenPrefix string) error {
	ctx := context.Background()
	senderID := c.Sender().ID

	// Find full token
	bots, err := f.repo.GetBotsByOwner(ctx, senderID)
	if err != nil {
		return c.Respond(&telebot.CallbackResponse{Text: "Error!", ShowAlert: true})
	}

	var fullToken string
	var ownerID int64
	var botID int64
	for _, bot := range bots {
		if strings.HasPrefix(bot.Token, tokenPrefix) {
			fullToken = bot.Token
			ownerID = bot.OwnerChatID
			botID = bot.ID
			break
		}
	}

	if fullToken == "" {
		return c.Respond(&telebot.CallbackResponse{Text: "Bot not found!", ShowAlert: true})
	}

	// Activate in database
	if err := f.repo.ActivateBot(ctx, fullToken); err != nil {
		return c.Respond(&telebot.CallbackResponse{Text: "Failed to activate!", ShowAlert: true})
	}

	// Start the bot
	if err := f.manager.StartBot(fullToken, ownerID, botID); err != nil {
		return c.Respond(&telebot.CallbackResponse{Text: "Failed to start: " + err.Error(), ShowAlert: true})
	}

	c.Respond(&telebot.CallbackResponse{Text: "✅ Bot started!"})
	return f.handleBotDetails(c, tokenPrefix)
}

// handleStopBotAction stops a bot
func (f *Factory) handleStopBotAction(c telebot.Context, tokenPrefix string) error {
	ctx := context.Background()
	senderID := c.Sender().ID

	// Find full token
	bots, err := f.repo.GetBotsByOwner(ctx, senderID)
	if err != nil {
		return c.Respond(&telebot.CallbackResponse{Text: "Error!", ShowAlert: true})
	}

	var fullToken string
	for _, bot := range bots {
		if strings.HasPrefix(bot.Token, tokenPrefix) {
			fullToken = bot.Token
			break
		}
	}

	if fullToken == "" {
		return c.Respond(&telebot.CallbackResponse{Text: "Bot not found!", ShowAlert: true})
	}

	// Deactivate in database
	f.repo.DeactivateBot(ctx, fullToken)

	// Stop the bot
	f.manager.StopBot(fullToken)

	c.Respond(&telebot.CallbackResponse{Text: "✅ Bot stopped!"})
	return f.handleBotDetails(c, tokenPrefix)
}

// handleDeleteBotConfirm shows delete confirmation
func (f *Factory) handleDeleteBotConfirm(c telebot.Context, tokenPrefix string) error {
	msg := `⚠️ <b>Confirm Deletion</b>

Are you sure you want to delete this bot?
This action cannot be undone!`

	menu := &telebot.ReplyMarkup{}
	btnConfirm := menu.Data("✅ Yes, Delete", CallbackConfirmDel, tokenPrefix)
	btnCancel := menu.Data("❌ Cancel", CallbackCancelDel)

	menu.Inline(
		menu.Row(btnConfirm, btnCancel),
	)

	return c.Edit(msg, menu, telebot.ModeHTML)
}

// handleConfirmDelete actually deletes the bot
func (f *Factory) handleConfirmDelete(c telebot.Context, tokenPrefix string) error {
	ctx := context.Background()
	senderID := c.Sender().ID

	// Find full token
	bots, err := f.repo.GetBotsByOwner(ctx, senderID)
	if err != nil {
		return c.Respond(&telebot.CallbackResponse{Text: "Error!", ShowAlert: true})
	}

	var fullToken string
	for _, bot := range bots {
		if strings.HasPrefix(bot.Token, tokenPrefix) {
			fullToken = bot.Token
			break
		}
	}

	if fullToken == "" {
		return c.Respond(&telebot.CallbackResponse{Text: "Bot not found!", ShowAlert: true})
	}

	// Stop if running
	f.manager.StopBot(fullToken)

	// Delete from database
	if err := f.repo.DeleteBot(ctx, fullToken); err != nil {
		return c.Respond(&telebot.CallbackResponse{Text: "Failed to delete!", ShowAlert: true})
	}

	c.Respond(&telebot.CallbackResponse{Text: "✅ Bot deleted!"})

	// Return to my bots list
	return f.handleMyBotsBtn(c)
}

// handleCancelDeleteBtn cancels deletion and returns to my bots
func (f *Factory) handleCancelDeleteBtn(c telebot.Context) error {
	c.Respond(&telebot.CallbackResponse{Text: "Cancelled"})
	return f.handleMyBotsBtn(c)
}

// handleBotSelectBtn handles bot selection from list
func (f *Factory) handleBotSelectBtn(c telebot.Context) error {
	tokenPrefix := c.Callback().Data
	log.Printf("[DEBUG] handleBotSelectBtn called - Unique: %s, Data: %s", c.Callback().Unique, tokenPrefix)
	return f.handleBotDetails(c, tokenPrefix)
}

// handleStartBotBtn handles start bot button
func (f *Factory) handleStartBotBtn(c telebot.Context) error {
	tokenPrefix := c.Callback().Data
	return f.handleStartBotAction(c, tokenPrefix)
}

// handleStopBotBtn handles stop bot button
func (f *Factory) handleStopBotBtn(c telebot.Context) error {
	tokenPrefix := c.Callback().Data
	return f.handleStopBotAction(c, tokenPrefix)
}

// handleDeleteBotBtn handles delete bot button
func (f *Factory) handleDeleteBotBtn(c telebot.Context) error {
	tokenPrefix := c.Callback().Data
	return f.handleDeleteBotConfirm(c, tokenPrefix)
}

// handleConfirmDelBtn handles confirm delete button
func (f *Factory) handleConfirmDelBtn(c telebot.Context) error {
	tokenPrefix := c.Callback().Data
	return f.handleConfirmDelete(c, tokenPrefix)
}

// handleStatsBtn shows system stats (admin only)
func (f *Factory) handleStatsBtn(c telebot.Context) error {
	if c.Sender().ID != f.adminID {
		return c.Respond(&telebot.CallbackResponse{Text: "Admin only!", ShowAlert: true})
	}

	ctx := context.Background()

	// Get all non-deleted bots
	bots, err := f.repo.GetAllBots(ctx)
	if err != nil {
		return c.Edit("❌ Failed to get stats.", f.getBackButton())
	}

	// Get deleted bots count
	deletedCount, err := f.repo.GetDeletedBotsCount(ctx)
	if err != nil {
		log.Printf("Failed to get deleted bots count: %v", err)
		deletedCount = 0
	}

	// Count running bots
	runningCount := 0
	for _, bot := range bots {
		if f.manager.IsRunning(bot.Token) {
			runningCount++
		}
	}

	// Get unique bot owners count
	ownerCount, _ := f.repo.GetUniqueOwnerCount(ctx)

	// Get user statistics
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	totalUsers, _ := f.repo.GetGlobalUniqueUserCount(ctx)
	activeUsers24h, _ := f.repo.GetGlobalActiveUserCount(ctx, now.AddDate(0, 0, -1))
	activeUsers7d, _ := f.repo.GetGlobalActiveUserCount(ctx, now.AddDate(0, 0, -7))
	newUsersToday, _ := f.repo.GetGlobalNewUserCount(ctx, todayStart)
	bannedCount, _ := f.repo.GetGlobalBannedUserCount(ctx)

	// Get message statistics
	totalMessages, _ := f.repo.GetGlobalTotalMessageCount(ctx)
	messagesToday, _ := f.repo.GetGlobalMessageCountSince(ctx, todayStart)
	messagesWeek, _ := f.repo.GetGlobalMessageCountSince(ctx, now.AddDate(0, 0, -7))

	// Get configuration statistics
	autoReplyCount, _ := f.repo.GetGlobalAutoReplyCount(ctx)
	forcedChannelCount, _ := f.repo.GetGlobalForcedChannelCount(ctx)

	msg := fmt.Sprintf(`📊 <b>System Statistics</b>

<b>🤖 Bots</b>
├ Total: %d
├ Running: %d
├ Stopped: %d
├ Deleted: %d
└ Owners: %d

<b>👥 Users</b>
├ Total: %d
├ Active (24h): %d
├ Active (7d): %d
├ New today: %d
└ Banned: %d

<b>📨 Messages</b>
├ Total: %d
├ Today: %d
└ This week: %d

<b>⚙️ Configuration</b>
├ Auto-replies: %d
└ Forced channels: %d`,
		len(bots), runningCount, len(bots)-runningCount, deletedCount, ownerCount,
		totalUsers, activeUsers24h, activeUsers7d, newUsersToday, bannedCount,
		totalMessages, messagesToday, messagesWeek,
		autoReplyCount, forcedChannelCount)

	return c.Edit(msg, f.getBackButton(), telebot.ModeHTML)
}

// handleText processes text messages (mainly for token submission)
func (f *Factory) handleText(c telebot.Context) error {
	text := strings.TrimSpace(c.Text())

	// Check if it looks like a bot token
	if !isValidTokenFormat(text) {
		return nil // Ignore non-token messages
	}

	return f.processToken(c, text)
}

// processToken validates and adds a new bot
func (f *Factory) processToken(c telebot.Context, token string) error {
	ctx := context.Background()
	senderID := c.Sender().ID

	// Check if bot already exists (active)
	existingBot, err := f.repo.GetBotByToken(ctx, token)
	if err != nil {
		log.Printf("Error checking existing bot: %v", err)
		return c.Reply("❌ An error occurred. Please try again.", f.getBackButton())
	}

	if existingBot != nil {
		if existingBot.OwnerChatID == senderID {
			return c.Reply("⚠️ You have already added this bot!", f.getBackButton())
		}
		return c.Reply("❌ This bot is already registered by another user.", f.getBackButton())
	}

	// Validate the token by creating a test bot logic
	testSettings := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 1 * time.Second},
	}

	testBot, err := telebot.NewBot(testSettings)
	if err != nil {
		log.Printf("Invalid token submitted: %v", err)
		return c.Reply("❌ Invalid token! Please check your token and try again.", f.getBackButton())
	}

	botInfo := testBot.Me

	// Check if bot was previously deleted (soft delete) - restore it
	deletedBot, err := f.repo.GetDeletedBotByToken(ctx, token)
	if err != nil {
		log.Printf("Error checking deleted bot: %v", err)
	}

	var botID int64
	if deletedBot != nil {
		// Restore the deleted bot
		if err := f.repo.RestoreBot(ctx, token, senderID, botInfo.Username); err != nil {
			log.Printf("Failed to restore bot: %v", err)
			return c.Reply("❌ Failed to restore bot. Please try again.", f.getBackButton())
		}
		botID = deletedBot.ID
		log.Printf("Bot restored: %s (ID: %d)", botInfo.Username, botID)
	} else {
		// Create new bot
		savedBot, err := f.repo.CreateBot(ctx, token, senderID, botInfo.Username)
		if err != nil {
			log.Printf("Failed to save bot: %v", err)
			return c.Reply("❌ Failed to save bot. Please try again.", f.getBackButton())
		}
		botID = savedBot.ID
	}

	// Delete the message containing the token for security
	if err := c.Bot().Delete(c.Message()); err != nil {
		log.Printf("Warning: Failed to delete token message: %v", err)
	}

	// Start the bot (Set Webhook)
	if err := f.manager.StartBot(token, senderID, botID); err != nil {
		log.Printf("Failed to start bot: %v", err)
		return c.Reply(fmt.Sprintf(`⚠️ Bot saved but failed to set webhook.

<b>Bot:</b> @%s
<b>Status:</b> Inactive (Webhook failed)

Go to My Bots to try starting it manually.`, botInfo.Username), f.getBackButton(), telebot.ModeHTML)
	}

	isAdmin := c.Sender().ID == f.adminID

	// Different message for restored vs new bot
	var successMsg string
	if deletedBot != nil {
		successMsg = fmt.Sprintf(`✅ <b>Bot Restored Successfully!</b>

<b>Bot:</b> @%s
<b>Name:</b> %s
<b>Status:</b> 🟢 Running (Webhook Set)

Your bot has been restored with all its previous data (messages, banned users, etc.)!`,
			botInfo.Username, botInfo.FirstName)
	} else {
		successMsg = fmt.Sprintf(`✅ <b>Bot Added Successfully!</b>

<b>Bot:</b> @%s
<b>Name:</b> %s
<b>Status:</b> 🟢 Running (Webhook Set)

Users can now message your bot and you'll receive their messages here!`,
			botInfo.Username, botInfo.FirstName)
	}

	return c.Reply(successMsg, f.getMainMenu(isAdmin), telebot.ModeHTML)
}
