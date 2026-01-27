package factory

import (
	"gopkg.in/telebot.v3"
)

// registerHandlers sets up all handlers for factory bot
func (f *Factory) registerHandlers() {
	// Only /start command - everything else is buttons
	f.bot.Handle("/start", f.handleStart)

	// Button callbacks (static)
	f.bot.Handle(&telebot.Btn{Unique: CallbackAddBot}, f.handleAddBotBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackMyBots}, f.handleMyBotsBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackStats}, f.handleStatsBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackMainMenu}, f.handleMainMenuBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackCancelDel}, f.handleCancelDeleteBtn)

	// Button callbacks (with data)
	f.bot.Handle(&telebot.Btn{Unique: CallbackBotSelect}, f.handleBotSelectBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackStartBot}, f.handleStartBotBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackStopBot}, f.handleStopBotBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackDeleteBot}, f.handleDeleteBotBtn)
	f.bot.Handle(&telebot.Btn{Unique: CallbackConfirmDel}, f.handleConfirmDelBtn)

	// Handle text messages (for token submission)
	f.bot.Handle(telebot.OnText, f.handleText)
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
