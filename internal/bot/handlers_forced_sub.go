package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Amr-9/botforge/internal/models"
	"gopkg.in/telebot.v3"
)

// checkForcedSubscription verifies user is subscribed to all required channels
// Returns: (isSubscribed bool, menu *telebot.ReplyMarkup, blockedMessage string, error)
func (m *Manager) checkForcedSubscription(ctx context.Context, c telebot.Context, bot *telebot.Bot,
	token string, botID int64, userID int64) (bool, *telebot.ReplyMarkup, string, error) {

	// Check if feature enabled (cache-first)
	enabled, cacheHit, err := m.cache.GetForcedSubEnabled(ctx, token)
	if err != nil {
		log.Printf("Error getting forced sub enabled from cache: %v", err)
	}

	if !cacheHit {
		botModel, err := m.repo.GetBotByToken(ctx, token)
		if err != nil {
			log.Printf("Error getting bot for forced sub check: %v", err)
			return true, nil, "", nil // Allow on error
		}
		if botModel != nil {
			enabled = botModel.ForcedSubEnabled
			m.cache.SetForcedSubEnabled(ctx, token, enabled)
		}
	}

	if !enabled {
		return true, nil, "", nil
	}

	// Check if user already verified recently
	if verified, _ := m.cache.IsUserSubVerified(ctx, token, userID); verified {
		return true, nil, "", nil
	}

	// Get required channels from DB
	channels, err := m.repo.GetForcedChannels(ctx, botID)
	if err != nil {
		log.Printf("Error getting forced channels: %v", err)
		return true, nil, "", nil // Allow on error
	}

	if len(channels) == 0 {
		return true, nil, "", nil
	}

	// Check subscription for each channel
	var notSubscribed []models.ForcedChannel

	for _, channel := range channels {
		member, err := bot.ChatMemberOf(&telebot.Chat{ID: channel.ChannelID}, &telebot.User{ID: userID})
		if err != nil {
			// Bot might not be admin anymore - log and skip this channel (lenient approach)
			log.Printf("Error checking membership for channel %d (bot may have lost admin): %v", channel.ChannelID, err)
			continue
		}

		// Check member status
		switch member.Role {
		case telebot.Creator, telebot.Administrator, telebot.Member:
			// User is subscribed
		default:
			// Not subscribed (left, kicked, restricted)
			notSubscribed = append(notSubscribed, channel)
		}
	}

	if len(notSubscribed) == 0 {
		// All subscribed, cache verification
		m.cache.SetUserSubVerified(ctx, token, userID)
		return true, nil, "", nil
	}

	// Build blocked message with join buttons
	menu, blockedMsg := m.buildSubscriptionRequiredMessage(ctx, token, notSubscribed)
	return false, menu, blockedMsg, nil
}

// buildSubscriptionRequiredMessage creates the message and buttons for non-subscribers
func (m *Manager) buildSubscriptionRequiredMessage(ctx context.Context, token string, channels []models.ForcedChannel) (*telebot.ReplyMarkup, string) {
	// Get custom message if set
	botModel, _ := m.repo.GetBotByToken(ctx, token)
	customMsg := ""
	if botModel != nil && botModel.ForcedSubMessage != "" {
		customMsg = botModel.ForcedSubMessage
	}

	var msgBuilder strings.Builder
	msgBuilder.WriteString("🔐 <b>Subscription Required</b>\n\n")

	if customMsg != "" {
		msgBuilder.WriteString(customMsg)
		msgBuilder.WriteString("\n\n")
	} else {
		msgBuilder.WriteString("Please subscribe to the following channels to use this bot:\n\n")
	}

	// Build menu with join buttons
	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	for _, channel := range channels {
		// Determine the URL for the channel
		var joinURL string
		if channel.InviteLink != "" {
			joinURL = channel.InviteLink
		} else if channel.ChannelUsername != "" {
			joinURL = fmt.Sprintf("https://t.me/%s", strings.TrimPrefix(channel.ChannelUsername, "@"))
		} else {
			// No link available, skip this channel in buttons
			continue
		}

		title := channel.ChannelTitle
		if title == "" {
			title = "Channel"
		}
		btn := menu.URL(fmt.Sprintf("📺 %s", title), joinURL)
		rows = append(rows, menu.Row(btn))
	}

	// Add check subscription button
	btnCheck := menu.Data("✅ Check Subscription", "check_subscription")
	rows = append(rows, menu.Row(btnCheck))

	menu.Inline(rows...)

	return menu, msgBuilder.String()
}

// handleForcedSubMenu shows the forced subscription settings menu
func (m *Manager) handleForcedSubMenu(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Get bot settings
		botModel, err := m.repo.GetBotByToken(ctx, token)
		if err != nil {
			log.Printf("Error getting bot for forced sub menu: %v", err)
			return c.Respond(&telebot.CallbackResponse{Text: "Error loading settings", ShowAlert: true})
		}

		enabled := false
		if botModel != nil {
			enabled = botModel.ForcedSubEnabled
		}

		// Get channels
		channels, err := m.repo.GetForcedChannels(ctx, botID)
		if err != nil {
			log.Printf("Error getting forced channels: %v", err)
		}

		// Build message
		var msgBuilder strings.Builder
		msgBuilder.WriteString("🔐 <b>Forced Subscription Settings</b>\n\n")

		statusText := "❌ Disabled"
		if enabled {
			statusText = "✅ Enabled"
		}
		msgBuilder.WriteString(fmt.Sprintf("<b>Status:</b> %s\n\n", statusText))

		if len(channels) > 0 {
			msgBuilder.WriteString(fmt.Sprintf("<b>Required Channels (%d):</b>\n", len(channels)))
			for i, ch := range channels {
				prefix := "├"
				if i == len(channels)-1 {
					prefix = "└"
				}
				title := ch.ChannelTitle
				if title == "" {
					title = fmt.Sprintf("Channel %d", ch.ChannelID)
				}
				msgBuilder.WriteString(fmt.Sprintf("%s 📺 %s\n", prefix, title))
			}
		} else {
			msgBuilder.WriteString("<i>No channels configured</i>\n")
		}

		// Build menu
		menu := &telebot.ReplyMarkup{}

		// Toggle button
		toggleText := "✅ Enable"
		if enabled {
			toggleText = "❌ Disable"
		}
		btnToggle := menu.Data(toggleText, "toggle_forced_sub")

		btnAddChannel := menu.Data("➕ Add Channel", "add_forced_channel")
		btnListChannels := menu.Data(fmt.Sprintf("📋 Manage Channels (%d)", len(channels)), "list_forced_channels")
		btnSetMessage := menu.Data("📝 Set Custom Message", "set_forced_sub_msg")
		btnBack := menu.Data("« Back to Settings", "back_to_settings")

		menu.Inline(
			menu.Row(btnToggle),
			menu.Row(btnAddChannel),
			menu.Row(btnListChannels),
			menu.Row(btnSetMessage),
			menu.Row(btnBack),
		)

		return c.Edit(msgBuilder.String(), menu, telebot.ModeHTML)
	}
}

// handleToggleForcedSub toggles the forced subscription feature on/off
func (m *Manager) handleToggleForcedSub(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Get current state
		botModel, err := m.repo.GetBotByToken(ctx, token)
		if err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Error loading settings", ShowAlert: true})
		}

		newState := true
		if botModel != nil && botModel.ForcedSubEnabled {
			newState = false
		}

		// Update in DB
		if err := m.repo.UpdateForcedSubEnabled(ctx, botID, newState); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Error updating settings", ShowAlert: true})
		}

		// Invalidate cache
		m.cache.InvalidateForcedSubEnabled(ctx, token)

		// Show feedback
		msg := "Forced subscription disabled"
		if newState {
			msg = "Forced subscription enabled"
		}
		c.Respond(&telebot.CallbackResponse{Text: msg})

		// Refresh menu
		return m.handleForcedSubMenu(bot, token, ownerChat)(c)
	}
}

// handleAddForcedChannel initiates the add channel flow
func (m *Manager) handleAddForcedChannel(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		if err := m.cache.SetUserState(ctx, token, c.Sender().ID, "add_forced_channel"); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Error setting state", ShowAlert: true})
		}

		menu := &telebot.ReplyMarkup{}
		btnCancel := menu.Data("❌ Cancel", "forced_sub_menu")
		menu.Inline(menu.Row(btnCancel))

		msg := `➕ <b>Add Forced Subscription Channel</b>

<b>Step 1:</b> Make the bot an <b>admin</b> in your channel first

<b>Step 2:</b> Send the channel info using one of these methods:
• Forward any message from the channel
• Send the channel username (e.g., <code>@channelname</code>)

<i>Note: For private channels, forward a message from the channel.</i>`

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// processAddForcedChannel processes the channel input
func (m *Manager) processAddForcedChannel(ctx context.Context, c telebot.Context, bot *telebot.Bot, token string) error {
	m.mu.RLock()
	botID := m.botIDs[token]
	m.mu.RUnlock()

	var channelID int64
	var channelUsername string
	var channelTitle string
	var inviteLink string

	// Check if it's a forwarded message from a channel
	if c.Message().SenderChat != nil && c.Message().SenderChat.Type == telebot.ChatChannel {
		chat := c.Message().SenderChat
		channelID = chat.ID
		channelUsername = chat.Username
		channelTitle = chat.Title
	} else {
		// Try to parse as username
		text := strings.TrimSpace(c.Text())
		if text == "" {
			return c.Reply("Please forward a message from the channel or send the channel username.")
		}

		// Remove @ if present
		username := strings.TrimPrefix(text, "@")

		// Try to get chat info
		chat, err := bot.ChatByUsername(username)
		if err != nil {
			return c.Reply("❌ Channel not found. Please check the username or forward a message from the channel.")
		}

		if chat.Type != telebot.ChatChannel {
			return c.Reply("❌ This is not a channel. Please provide a channel username.")
		}

		channelID = chat.ID
		channelUsername = chat.Username
		channelTitle = chat.Title
	}

	// Check if bot is admin in the channel
	botMember, err := bot.ChatMemberOf(&telebot.Chat{ID: channelID}, bot.Me)
	if err != nil {
		m.cache.ClearUserState(ctx, token, c.Sender().ID)
		return c.Reply("❌ Cannot access this channel. Make sure the bot is added as an admin.")
	}

	if botMember.Role != telebot.Administrator && botMember.Role != telebot.Creator {
		m.cache.ClearUserState(ctx, token, c.Sender().ID)
		return c.Reply("❌ Bot must be an admin in the channel to check subscriptions.")
	}

	// Check if channel already exists
	existing, _ := m.repo.GetForcedChannel(ctx, botID, channelID)
	if existing != nil {
		m.cache.ClearUserState(ctx, token, c.Sender().ID)
		return c.Reply("⚠️ This channel is already in the list.")
	}

	// For private channels, try to get invite link
	if channelUsername == "" {
		// Try to get or create invite link
		chat, err := bot.ChatByID(channelID)
		if err == nil && chat.InviteLink != "" {
			inviteLink = chat.InviteLink
		}
	}

	// Save to database
	if err := m.repo.CreateForcedChannel(ctx, botID, channelID, channelUsername, channelTitle, inviteLink); err != nil {
		m.cache.ClearUserState(ctx, token, c.Sender().ID)
		return c.Reply("❌ Failed to add channel. Please try again.")
	}

	// Clear all user subscription verifications (since channel list changed)
	m.cache.ClearAllUserSubVerified(ctx, token)

	// Clear state
	m.cache.ClearUserState(ctx, token, c.Sender().ID)

	// Success message
	msg := fmt.Sprintf("✅ Channel <b>%s</b> added successfully!", channelTitle)

	menu := &telebot.ReplyMarkup{}
	btnBack := menu.Data("« Back to Forced Sub Settings", "forced_sub_menu")
	menu.Inline(menu.Row(btnBack))

	return c.Reply(msg, menu, telebot.ModeHTML)
}

// handleListForcedChannels shows list of configured channels with remove option
func (m *Manager) handleListForcedChannels(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		channels, err := m.repo.GetForcedChannels(ctx, botID)
		if err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Error loading channels", ShowAlert: true})
		}

		if len(channels) == 0 {
			menu := &telebot.ReplyMarkup{}
			btnAdd := menu.Data("➕ Add Channel", "add_forced_channel")
			btnBack := menu.Data("« Back", "forced_sub_menu")
			menu.Inline(menu.Row(btnAdd), menu.Row(btnBack))
			return c.Edit("📋 <b>Forced Subscription Channels</b>\n\n<i>No channels configured yet.</i>", menu, telebot.ModeHTML)
		}

		var msgBuilder strings.Builder
		msgBuilder.WriteString("📋 <b>Forced Subscription Channels</b>\n\n")
		msgBuilder.WriteString("Click on a channel to remove it:\n\n")

		menu := &telebot.ReplyMarkup{}
		var rows []telebot.Row

		for _, ch := range channels {
			title := ch.ChannelTitle
			if title == "" {
				title = fmt.Sprintf("Channel %d", ch.ChannelID)
			}
			btn := menu.Data(fmt.Sprintf("❌ %s", title), "del_forced_channel", strconv.FormatInt(ch.ChannelID, 10))
			rows = append(rows, menu.Row(btn))
		}

		btnAdd := menu.Data("➕ Add Channel", "add_forced_channel")
		btnBack := menu.Data("« Back", "forced_sub_menu")
		rows = append(rows, menu.Row(btnAdd))
		rows = append(rows, menu.Row(btnBack))

		menu.Inline(rows...)

		return c.Edit(msgBuilder.String(), menu, telebot.ModeHTML)
	}
}

// handleRemoveForcedChannel removes a channel from the list
func (m *Manager) handleRemoveForcedChannel(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		// Get channel ID from callback data
		data := c.Callback().Data
		// Data format: "del_forced_channel|<channel_id>"
		parts := strings.Split(data, "|")
		if len(parts) < 2 {
			return c.Respond(&telebot.CallbackResponse{Text: "Invalid data", ShowAlert: true})
		}

		channelID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Invalid channel ID", ShowAlert: true})
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Delete from DB
		if err := m.repo.DeleteForcedChannel(ctx, botID, channelID); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Error removing channel", ShowAlert: true})
		}

		// Clear all user subscription verifications
		m.cache.ClearAllUserSubVerified(ctx, token)

		c.Respond(&telebot.CallbackResponse{Text: "Channel removed"})

		// Refresh list
		return m.handleListForcedChannels(bot, token, ownerChat)(c)
	}
}

// handleSetForcedSubMsg initiates custom message setting flow
func (m *Manager) handleSetForcedSubMsg(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		if err := m.cache.SetUserState(ctx, token, c.Sender().ID, "set_forced_sub_message"); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Error setting state", ShowAlert: true})
		}

		// Get current message
		botModel, _ := m.repo.GetBotByToken(ctx, token)
		currentMsg := "<i>(Default message)</i>"
		if botModel != nil && botModel.ForcedSubMessage != "" {
			currentMsg = strings.ReplaceAll(botModel.ForcedSubMessage, "<", "&lt;")
			currentMsg = strings.ReplaceAll(currentMsg, ">", "&gt;")
		}

		menu := &telebot.ReplyMarkup{}
		btnClear := menu.Data("🗑️ Clear (Use Default)", "clear_forced_sub_msg")
		btnCancel := menu.Data("❌ Cancel", "forced_sub_menu")
		menu.Inline(menu.Row(btnClear), menu.Row(btnCancel))

		msg := fmt.Sprintf(`📝 <b>Set Custom Message</b>

<b>Current Message:</b>
<pre>%s</pre>

Send the message that will be shown to users who haven't subscribed yet.

This message will appear above the channel join buttons.`, currentMsg)

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// processSetForcedSubMessage saves the custom message
func (m *Manager) processSetForcedSubMessage(ctx context.Context, c telebot.Context, token string) error {
	m.mu.RLock()
	botID := m.botIDs[token]
	m.mu.RUnlock()

	message := strings.TrimSpace(c.Text())
	if message == "" {
		return c.Reply("Please send a message text.")
	}

	// Save to database
	if err := m.repo.UpdateForcedSubMessage(ctx, botID, message); err != nil {
		m.cache.ClearUserState(ctx, token, c.Sender().ID)
		return c.Reply("❌ Failed to save message. Please try again.")
	}

	// Clear state
	m.cache.ClearUserState(ctx, token, c.Sender().ID)

	menu := &telebot.ReplyMarkup{}
	btnBack := menu.Data("« Back to Forced Sub Settings", "forced_sub_menu")
	menu.Inline(menu.Row(btnBack))

	return c.Reply("✅ Custom message saved successfully!", menu)
}

// handleClearForcedSubMsg clears the custom message
func (m *Manager) handleClearForcedSubMsg(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Clear in database
		if err := m.repo.UpdateForcedSubMessage(ctx, botID, ""); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Error clearing message", ShowAlert: true})
		}

		// Clear state if any
		m.cache.ClearUserState(ctx, token, c.Sender().ID)

		c.Respond(&telebot.CallbackResponse{Text: "Message cleared, using default"})

		// Back to menu
		return m.handleForcedSubMenu(bot, token, ownerChat)(c)
	}
}

// handleCheckSubscription handles the "Check Subscription" button from users
func (m *Manager) handleCheckSubscription(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		ctx := context.Background()
		userID := c.Sender().ID

		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Clear previous verification cache to force re-check
		m.cache.ClearUserSubVerified(ctx, token, userID)

		// Check subscription status
		isSubscribed, menu, blockedMsg, err := m.checkForcedSubscription(ctx, c, bot, token, botID, userID)
		if err != nil {
			log.Printf("Error checking subscription: %v", err)
		}

		if isSubscribed {
			// User is now subscribed
			c.Respond(&telebot.CallbackResponse{Text: "✅ Subscription verified! You can now use the bot.", ShowAlert: true})

			// Show welcome message
			botModel, _ := m.repo.GetBotByToken(ctx, token)
			welcomeMsg := "👋 Welcome! You can now send me your message."
			if botModel != nil && botModel.StartMessage != "" {
				welcomeMsg = botModel.StartMessage
			}
			return c.Edit(welcomeMsg, telebot.ModeMarkdown)
		}

		// Still not subscribed
		c.Respond(&telebot.CallbackResponse{Text: "❌ You haven't subscribed to all required channels yet.", ShowAlert: true})
		return c.Edit(blockedMsg, menu, telebot.ModeHTML)
	}
}

// processForcedSubState processes multi-step flow states for forced subscription
func (m *Manager) processForcedSubState(ctx context.Context, c telebot.Context, bot *telebot.Bot, token string, state string) (bool, error) {
	switch state {
	case "add_forced_channel":
		return true, m.processAddForcedChannel(ctx, c, bot, token)
	case "set_forced_sub_message":
		return true, m.processSetForcedSubMessage(ctx, c, token)
	}
	return false, nil
}
