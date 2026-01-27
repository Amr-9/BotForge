package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"gopkg.in/telebot.v3"
)

// handleAutoRepliesMenu shows the auto-replies management menu
func (m *Manager) handleAutoRepliesMenu(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Get counts
		keywordCount, _ := m.repo.GetAutoReplyCount(ctx, botID, "keyword")
		commandCount, _ := m.repo.GetAutoReplyCount(ctx, botID, "command")

		// Get current forward setting
		botModel, _ := m.repo.GetBotByToken(ctx, token)
		forwardEnabled := true
		if botModel != nil {
			forwardEnabled = botModel.ForwardAutoReplies
		}

		// Forward toggle button text
		forwardBtnText := "📩 Forward to Admin: ON"
		if !forwardEnabled {
			forwardBtnText = "📩 Forward to Admin: OFF"
		}

		menu := &telebot.ReplyMarkup{}
		btnAddKeyword := menu.Data("➕ Add Auto-Reply", "add_auto_reply")
		btnAddCommand := menu.Data("➕ Add Command", "add_custom_cmd")
		btnListKeywords := menu.Data(fmt.Sprintf("📋 Auto-Replies (%d)", keywordCount), "list_auto_replies")
		btnListCommands := menu.Data(fmt.Sprintf("📋 Commands (%d)", commandCount), "list_custom_cmds")
		btnToggleForward := menu.Data(forwardBtnText, "toggle_forward_replies")
		btnBack := menu.Data("« Back", "child_settings")

		menu.Inline(
			menu.Row(btnAddKeyword, btnAddCommand),
			menu.Row(btnListKeywords),
			menu.Row(btnListCommands),
			menu.Row(btnToggleForward),
			menu.Row(btnBack),
		)

		forwardStatus := "✅ ON - Auto-replied messages are forwarded to you"
		if !forwardEnabled {
			forwardStatus = "❌ OFF - Auto-replied messages are NOT forwarded"
		}

		msg := fmt.Sprintf(`🤖 <b>Auto-Replies & Custom Commands</b>

<b>📍 Auto-Replies:</b> Respond to specific keywords (exact match)
<b>📍 Custom Commands:</b> Respond to commands like /help

<b>📩 Forward Setting:</b>
%s

✅ Supports Markdown formatting`, forwardStatus)

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleToggleForwardReplies toggles the forward_auto_replies setting
func (m *Manager) handleToggleForwardReplies(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Get current setting
		botModel, err := m.repo.GetBotByToken(ctx, token)
		if err != nil || botModel == nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Error getting bot settings", ShowAlert: true})
		}

		// Toggle the setting
		newValue := !botModel.ForwardAutoReplies
		if err := m.repo.UpdateBotForwardAutoReplies(ctx, botID, newValue); err != nil {
			log.Printf("Error updating forward_auto_replies: %v", err)
			return c.Respond(&telebot.CallbackResponse{Text: "Error updating setting", ShowAlert: true})
		}

		status := "ON ✅"
		if !newValue {
			status = "OFF ❌"
		}
		c.Respond(&telebot.CallbackResponse{Text: fmt.Sprintf("Forward to Admin: %s", status)})

		// Reload the menu to show updated status
		return m.handleAutoRepliesMenu(bot, token, ownerChat)(c)
	}
}

// handleAddAutoReply starts the flow to add a new auto-reply keyword
func (m *Manager) handleAddAutoReply(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		if err := m.cache.SetUserState(ctx, token, c.Sender().ID, "add_auto_reply_trigger"); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "An error occurred!", ShowAlert: true})
		}

		menu := &telebot.ReplyMarkup{}
		btnCancel := menu.Data("❌ Cancel", "auto_replies_menu")
		menu.Inline(menu.Row(btnCancel))

		msg := `➕ <b>Add Auto-Reply</b>

Send the trigger keyword that the bot will respond to automatically.

<b>Example:</b> <code>price</code> or <code>hello</code>

💡 The bot will respond if the keyword is found anywhere in the message.`

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleAddCustomCommand starts the flow to add a new custom command
func (m *Manager) handleAddCustomCommand(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		if err := m.cache.SetUserState(ctx, token, c.Sender().ID, "add_custom_cmd_name"); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "An error occurred!", ShowAlert: true})
		}

		menu := &telebot.ReplyMarkup{}
		btnCancel := menu.Data("❌ Cancel", "auto_replies_menu")
		menu.Inline(menu.Row(btnCancel))

		msg := `➕ <b>Add Custom Command</b>

Send the command name (without /).

<b>Example:</b> <code>help</code> or <code>about</code>

💡 Users will type <code>/help</code> to trigger the command.`

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleListAutoReplies shows a paginated list of keyword auto-replies
func (m *Manager) handleListAutoReplies(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		replies, err := m.repo.GetAutoReplies(ctx, botID, "keyword")
		if err != nil {
			log.Printf("Error getting auto-replies: %v", err)
			return c.Respond(&telebot.CallbackResponse{Text: "Error fetching data", ShowAlert: true})
		}

		menu := &telebot.ReplyMarkup{}

		if len(replies) == 0 {
			btnBack := menu.Data("« Back", "auto_replies_menu")
			menu.Inline(menu.Row(btnBack))
			return c.Edit("📋 <b>Auto-Replies</b>\n\n<i>No auto-replies yet.</i>", menu, telebot.ModeHTML)
		}

		var rows []telebot.Row
		for _, r := range replies {
			// Truncate long triggers for button display
			displayTrigger := r.TriggerWord
			if len(displayTrigger) > 20 {
				displayTrigger = displayTrigger[:17] + "..."
			}
			btn := menu.Data(fmt.Sprintf("🗑 %s", displayTrigger), "del_reply", fmt.Sprintf("%d", r.ID))
			rows = append(rows, menu.Row(btn))
		}

		btnBack := menu.Data("« Back", "auto_replies_menu")
		rows = append(rows, menu.Row(btnBack))
		menu.Inline(rows...)

		msg := fmt.Sprintf("📋 <b>Auto-Replies</b> (%d)\n\nTap a reply to delete it:", len(replies))
		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleListCustomCommands shows a paginated list of custom commands
func (m *Manager) handleListCustomCommands(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		commands, err := m.repo.GetAutoReplies(ctx, botID, "command")
		if err != nil {
			log.Printf("Error getting custom commands: %v", err)
			return c.Respond(&telebot.CallbackResponse{Text: "Error fetching data", ShowAlert: true})
		}

		menu := &telebot.ReplyMarkup{}

		if len(commands) == 0 {
			btnBack := menu.Data("« Back", "auto_replies_menu")
			menu.Inline(menu.Row(btnBack))
			return c.Edit("📋 <b>Custom Commands</b>\n\n<i>No custom commands yet.</i>", menu, telebot.ModeHTML)
		}

		var rows []telebot.Row
		for _, cmd := range commands {
			btn := menu.Data(fmt.Sprintf("🗑 /%s", cmd.TriggerWord), "del_reply", fmt.Sprintf("%d", cmd.ID))
			rows = append(rows, menu.Row(btn))
		}

		btnBack := menu.Data("« Back", "auto_replies_menu")
		rows = append(rows, menu.Row(btnBack))
		menu.Inline(rows...)

		msg := fmt.Sprintf("📋 <b>Custom Commands</b> (%d)\n\nTap a command to delete it:", len(commands))
		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleDeleteAutoReply deletes an auto-reply or custom command by ID
func (m *Manager) handleDeleteAutoReply(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Get ID from callback data
		data := c.Callback().Data
		var replyID int64
		if _, err := fmt.Sscanf(data, "%d", &replyID); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Invalid data", ShowAlert: true})
		}

		// Get the reply first to know its type (for cache invalidation)
		reply, err := m.repo.GetAutoReplyByID(ctx, replyID)
		if err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Reply not found", ShowAlert: true})
		}

		// Delete from DB
		if err := m.repo.DeleteAutoReply(ctx, botID, replyID); err != nil {
			log.Printf("Error deleting auto-reply: %v", err)
			return c.Respond(&telebot.CallbackResponse{Text: "Error deleting", ShowAlert: true})
		}

		// Invalidate cache
		m.cache.DeleteAutoReply(ctx, token, reply.TriggerWord, reply.TriggerType)

		c.Respond(&telebot.CallbackResponse{Text: "✅ Deleted successfully"})

		// Reload the appropriate list
		if reply.TriggerType == "command" {
			return m.handleListCustomCommands(bot, token, ownerChat)(c)
		}
		return m.handleListAutoReplies(bot, token, ownerChat)(c)
	}
}

// processAutoReplyState handles the multi-step flow for adding auto-replies
func (m *Manager) processAutoReplyState(ctx context.Context, c telebot.Context, token string, state string) (bool, error) {
	sender := c.Sender()
	text := strings.TrimSpace(c.Text())

	m.mu.RLock()
	botID := m.botIDs[token]
	m.mu.RUnlock()

	switch state {
	case "add_auto_reply_trigger":
		// Store trigger word temporarily and ask for response
		if text == "" {
			return true, c.Reply("⚠️ Please send a text message.")
		}

		// Check if trigger already exists
		existing, _ := m.repo.GetAutoReplyByTrigger(ctx, botID, text, "keyword")
		if existing != nil {
			return true, c.Reply("⚠️ This keyword already exists. Send a different one:")
		}

		// Store trigger temporarily
		m.cache.SetTempData(ctx, token, sender.ID, "trigger", text)
		m.cache.SetUserState(ctx, token, sender.ID, "add_auto_reply_response")

		menu := &telebot.ReplyMarkup{}
		btnCancel := menu.Data("❌ Cancel", "auto_replies_menu")
		menu.Inline(menu.Row(btnCancel))

		return true, c.Send(fmt.Sprintf("✅ Keyword: <code>%s</code>\n\nNow send the auto-reply response.\n\n💡 Supports Markdown formatting like:\n<code>*bold* and _italic_</code>", text), menu, telebot.ModeHTML)

	case "add_auto_reply_response":
		if text == "" {
			return true, c.Reply("⚠️ Please send a text response.")
		}

		// Get trigger from temp storage
		trigger, _ := m.cache.GetTempData(ctx, token, sender.ID, "trigger")
		if trigger == "" {
			m.cache.ClearUserState(ctx, token, sender.ID)
			return true, c.Reply("⚠️ Session expired. Please try again.")
		}

		// Save to DB
		err := m.repo.CreateAutoReply(ctx, botID, trigger, text, "keyword", "contains")
		if err != nil {
			log.Printf("Error creating auto-reply: %v", err)
			return true, c.Reply("❌ Error saving.")
		}

		// Cache for fast lookup
		m.cache.SetAutoReply(ctx, token, trigger, text, "keyword")

		// Clear state
		m.cache.ClearUserState(ctx, token, sender.ID)
		m.cache.ClearTempData(ctx, token, sender.ID, "trigger")

		return true, c.Reply(fmt.Sprintf("✅ <b>Auto-reply added!</b>\n\n🔑 Keyword: <code>%s</code>\n💬 Response: %s", trigger, text), telebot.ModeHTML)

	case "add_custom_cmd_name":
		if text == "" {
			return true, c.Reply("⚠️ Please send the command name.")
		}

		// Clean command name (remove / if present)
		cmdName := strings.TrimPrefix(text, "/")
		cmdName = strings.ToLower(cmdName)

		// Validate command name (alphanumeric only)
		for _, r := range cmdName {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
				return true, c.Reply("⚠️ Command name must contain only English letters, numbers, and underscores.")
			}
		}

		// Check if command already exists
		existing, _ := m.repo.GetAutoReplyByTrigger(ctx, botID, cmdName, "command")
		if existing != nil {
			return true, c.Reply("⚠️ This command already exists. Send a different one:")
		}

		// Store command name temporarily
		m.cache.SetTempData(ctx, token, sender.ID, "command", cmdName)
		m.cache.SetUserState(ctx, token, sender.ID, "add_custom_cmd_response")

		menu := &telebot.ReplyMarkup{}
		btnCancel := menu.Data("❌ Cancel", "auto_replies_menu")
		menu.Inline(menu.Row(btnCancel))

		return true, c.Send(fmt.Sprintf("✅ Command: <code>/%s</code>\n\nNow send the response for this command.\n\n💡 Supports Markdown formatting", cmdName), menu, telebot.ModeHTML)

	case "add_custom_cmd_response":
		if text == "" {
			return true, c.Reply("⚠️ Please send a text response.")
		}

		// Get command from temp storage
		cmdName, _ := m.cache.GetTempData(ctx, token, sender.ID, "command")
		if cmdName == "" {
			m.cache.ClearUserState(ctx, token, sender.ID)
			return true, c.Reply("⚠️ Session expired. Please try again.")
		}

		// Save to DB
		err := m.repo.CreateAutoReply(ctx, botID, cmdName, text, "command", "exact")
		if err != nil {
			log.Printf("Error creating custom command: %v", err)
			return true, c.Reply("❌ Error saving.")
		}

		// Cache for fast lookup
		m.cache.SetAutoReply(ctx, token, cmdName, text, "command")

		// Clear state
		m.cache.ClearUserState(ctx, token, sender.ID)
		m.cache.ClearTempData(ctx, token, sender.ID, "command")

		return true, c.Reply(fmt.Sprintf("✅ <b>Custom command added!</b>\n\n🔑 Command: <code>/%s</code>\n💬 Response: %s", cmdName, text), telebot.ModeHTML)
	}

	return false, nil
}

// checkAutoReply checks if a message matches any auto-reply triggers (exact match only)
func (m *Manager) checkAutoReply(ctx context.Context, token string, botID int64, text string) string {
	text = strings.ToLower(strings.TrimSpace(text))

	// Try cache first - get all keywords for this bot
	replies, err := m.cache.GetAllAutoReplies(ctx, token, "keyword")
	if err == nil && len(replies) > 0 {
		for trigger, response := range replies {
			if text == strings.ToLower(trigger) {
				return response
			}
		}
		return ""
	}

	// Fallback to DB
	dbReplies, err := m.repo.GetAutoReplies(ctx, botID, "keyword")
	if err != nil {
		log.Printf("Error getting auto-replies from DB: %v", err)
		return ""
	}

	for _, r := range dbReplies {
		if r.IsActive {
			trigger := strings.ToLower(r.TriggerWord)
			// Only exact match
			if text == trigger {
				// Cache for next time
				m.cache.SetAutoReply(ctx, token, r.TriggerWord, r.Response, "keyword")
				return r.Response
			}
		}
	}

	return ""
}

// checkCustomCommand checks if a message is a custom command
func (m *Manager) checkCustomCommand(ctx context.Context, token string, botID int64, text string) string {
	// Only check if it starts with /
	if !strings.HasPrefix(text, "/") {
		return ""
	}

	// Extract command name
	cmdText := strings.TrimPrefix(text, "/")
	cmdParts := strings.Fields(cmdText)
	if len(cmdParts) == 0 {
		return ""
	}
	cmdName := strings.ToLower(cmdParts[0])

	// Try cache first
	response, err := m.cache.GetAutoReply(ctx, token, cmdName, "command")
	if err == nil && response != "" {
		return response
	}

	// Fallback to DB
	reply, err := m.repo.GetAutoReplyByTrigger(ctx, botID, cmdName, "command")
	if err != nil || reply == nil || !reply.IsActive {
		return ""
	}

	// Cache for next time
	m.cache.SetAutoReply(ctx, token, cmdName, reply.Response, "command")
	return reply.Response
}
