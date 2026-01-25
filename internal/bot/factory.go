package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Amr-9/botforge/internal/database"
	"gopkg.in/telebot.v3"
)

// Button callback data constants
const (
	CallbackAddBot     = "add_bot"
	CallbackMyBots     = "my_bots"
	CallbackStats      = "stats"
	CallbackMainMenu   = "main_menu"
	CallbackBotPrefix  = "bot:"
	CallbackStartBot   = "start:"
	CallbackStopBot    = "stop:"
	CallbackDeleteBot  = "delete:"
	CallbackConfirmDel = "confirm_del:"
	CallbackCancelDel  = "cancel_del"
)

// Factory represents the main factory bot
type Factory struct {
	bot     *telebot.Bot
	repo    *database.Repository
	manager *Manager
	adminID int64
}

// NewFactory creates a new factory bot logic instance
// IMPORTANT: It now accepts an *existing* bot instance which is already configured with Webhook/Poller
func NewFactory(botInstance *telebot.Bot, repo *database.Repository, manager *Manager, adminID int64) (*Factory, error) {
	factory := &Factory{
		bot:     botInstance,
		repo:    repo,
		manager: manager,
		adminID: adminID,
	}

	factory.registerHandlers()

	return factory, nil
}

// GetBot returns the underlying bot instance
func (f *Factory) GetBot() *telebot.Bot {
	return f.bot
}

// registerHandlers sets up all handlers for factory bot
func (f *Factory) registerHandlers() {
	// Only /start command - everything else is buttons
	f.bot.Handle("/start", f.handleStart)

	// Button callbacks
	f.bot.Handle(&telebot.Btn{Unique: CallbackAddBot}, f.handleAddBotBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackMyBots}, f.handleMyBotsBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackStats}, f.handleStatsBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackMainMenu}, f.handleMainMenuBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackCancelDel}, f.handleCancelDeleteBtn)

	// Dynamic callbacks (with data)
	f.bot.Handle(telebot.OnCallback, f.handleDynamicCallback)

	// Handle text messages (for token submission)
	f.bot.Handle(telebot.OnText, f.handleText)
}

// Start starts the factory bot (No-op in Webhook mode as server drives it)
// But we keep it in case we switch back to Poller someday, or for logging
func (f *Factory) Start() {
	log.Println("Factory Bot Logic initialized.")
}

// Stop stops the factory bot
func (f *Factory) Stop() {
	// In webhook mode, we might want to remove webhook, but Manager handles StopAll
	// So we can just log here or do specific cleanup
	log.Println("Stopping Factory Bot logic...")
}

// getMainMenu returns the main menu inline keyboard
func (f *Factory) getMainMenu(isAdmin bool) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{}

	btnAddBot := menu.Data("➕ Add Bot", CallbackAddBot)
	btnMyBots := menu.Data("🤖 My Bots", CallbackMyBots)

	if isAdmin {
		btnStats := menu.Data("📊 Stats", CallbackStats)
		menu.Inline(
			menu.Row(btnAddBot),
			menu.Row(btnMyBots),
			menu.Row(btnStats),
		)
	} else {
		menu.Inline(
			menu.Row(btnAddBot),
			menu.Row(btnMyBots),
		)
	}

	return menu
}

// getBackButton returns a back to menu button
func (f *Factory) getBackButton() *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{}
	btnBack := menu.Data("« Back to Menu", CallbackMainMenu)
	menu.Inline(menu.Row(btnBack))
	return menu
}

// handleStart sends welcome message with main menu
func (f *Factory) handleStart(c telebot.Context) error {
	isAdmin := c.Sender().ID == f.adminID

	welcome := `🤖 <b>Welcome to Bot Factory!</b>

I can help you create and manage your own Telegram contact bots.

<b>How it works:</b>
1. Create a bot with @BotFather
2. Add it here using the button below
3. Users message your bot, you receive the messages here
4. Reply to forward your response back to them

Choose an option below:`

	return c.Send(welcome, f.getMainMenu(isAdmin), telebot.ModeHTML)
}

// handleMainMenuBtn returns to main menu
func (f *Factory) handleMainMenuBtn(c telebot.Context) error {
	isAdmin := c.Sender().ID == f.adminID

	welcome := `🤖 <b>Bot Factory - Main Menu</b>

Choose an option:`

	return c.Edit(welcome, f.getMainMenu(isAdmin), telebot.ModeHTML)
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

		// Show first 10 chars of token
		shortToken := bot.Token[:10] + "..."
		btnText := fmt.Sprintf("%s %s", status, shortToken)

		btn := menu.Data(btnText, CallbackBotPrefix+bot.Token[:20])
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
		token       string
		ownerChatID int64
	}

	for _, bot := range bots {
		if strings.HasPrefix(bot.Token, tokenPrefix) {
			targetBot = &struct {
				token       string
				ownerChatID int64
			}{token: bot.Token, ownerChatID: bot.OwnerChatID}
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

	msg := fmt.Sprintf(`🤖 <b>Bot Details</b>

<b>Token:</b> <code>%s</code>
<b>Status:</b> %s

Select an action:`, maskToken(targetBot.token), status)

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	if isRunning {
		btnStop := menu.Data("⏹ Stop Bot", CallbackStopBot+tokenPrefix)
		rows = append(rows, menu.Row(btnStop))
	} else {
		btnStart := menu.Data("▶️ Start Bot", CallbackStartBot+tokenPrefix)
		rows = append(rows, menu.Row(btnStart))
	}

	btnDelete := menu.Data("🗑 Delete Bot", CallbackDeleteBot+tokenPrefix)
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
	btnConfirm := menu.Data("✅ Yes, Delete", CallbackConfirmDel+tokenPrefix)
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

// handleDynamicCallback handles callbacks with dynamic data
func (f *Factory) handleDynamicCallback(c telebot.Context) error {
	data := c.Callback().Data

	switch {
	case strings.HasPrefix(data, CallbackBotPrefix):
		tokenPrefix := strings.TrimPrefix(data, CallbackBotPrefix)
		return f.handleBotDetails(c, tokenPrefix)

	case strings.HasPrefix(data, CallbackStartBot):
		tokenPrefix := strings.TrimPrefix(data, CallbackStartBot)
		return f.handleStartBotAction(c, tokenPrefix)

	case strings.HasPrefix(data, CallbackStopBot):
		tokenPrefix := strings.TrimPrefix(data, CallbackStopBot)
		return f.handleStopBotAction(c, tokenPrefix)

	case strings.HasPrefix(data, CallbackDeleteBot):
		tokenPrefix := strings.TrimPrefix(data, CallbackDeleteBot)
		return f.handleDeleteBotConfirm(c, tokenPrefix)

	case strings.HasPrefix(data, CallbackConfirmDel):
		tokenPrefix := strings.TrimPrefix(data, CallbackConfirmDel)
		return f.handleConfirmDelete(c, tokenPrefix)
	}

	return nil
}

// handleStatsBtn shows system stats (admin only)
func (f *Factory) handleStatsBtn(c telebot.Context) error {
	if c.Sender().ID != f.adminID {
		return c.Respond(&telebot.CallbackResponse{Text: "Admin only!", ShowAlert: true})
	}

	ctx := context.Background()

	// Get all bots
	bots, err := f.repo.GetActiveBots(ctx)
	if err != nil {
		return c.Edit("❌ Failed to get stats.", f.getBackButton())
	}

	runningCount := f.manager.GetRunningCount()

	msg := fmt.Sprintf(`📊 <b>System Statistics</b>

🤖 <b>Total Active Bots:</b> %d
🟢 <b>Running:</b> %d
🔴 <b>Stopped:</b> %d`,
		len(bots), runningCount, len(bots)-runningCount)

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

	// Check if bot already exists
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
	// Note: We use LongPoller temporarily just to validate against Telegram API
	// But we don't start it.
	testSettings := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 1 * time.Second},
	}

	testBot, err := telebot.NewBot(testSettings)
	if err != nil {
		log.Printf("Invalid token submitted: %v", err)
		return c.Reply("❌ Invalid token! Please check your token and try again.", f.getBackButton())
	}

	// Get bot info (makes a request to getMe)
	botInfo := testBot.Me

	// Save to database
	savedBot, err := f.repo.CreateBot(ctx, token, senderID)
	if err != nil {
		log.Printf("Failed to save bot: %v", err)
		return c.Reply("❌ Failed to save bot. Please try again.", f.getBackButton())
	}

	// Start the bot (Set Webhook)
	if err := f.manager.StartBot(token, senderID, savedBot.ID); err != nil {
		log.Printf("Failed to start bot: %v", err)
		return c.Reply(fmt.Sprintf(`⚠️ Bot saved but failed to set webhook.

<b>Bot:</b> @%s
<b>Status:</b> Inactive (Webhook failed)

Go to My Bots to try starting it manually.`, botInfo.Username), f.getBackButton(), telebot.ModeHTML)
	}

	isAdmin := c.Sender().ID == f.adminID

	return c.Reply(fmt.Sprintf(`✅ <b>Bot Added Successfully!</b>

<b>Bot:</b> @%s
<b>Name:</b> %s
<b>Status:</b> 🟢 Running (Webhook Set)

Users can now message your bot and you'll receive their messages here!`,
		botInfo.Username, botInfo.FirstName), f.getMainMenu(isAdmin), telebot.ModeHTML)
}

// isValidTokenFormat checks if a string looks like a bot token
func isValidTokenFormat(s string) bool {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return false
	}

	for _, c := range parts[0] {
		if c < '0' || c > '9' {
			return false
		}
	}

	if len(parts[1]) < 30 {
		return false
	}

	return true
}

// maskToken masks a token for display
func maskToken(token string) string {
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return "***"
	}

	if len(parts[1]) > 10 {
		return parts[0] + ":" + parts[1][:5] + "..." + parts[1][len(parts[1])-5:]
	}
	return parts[0] + ":***"
}
