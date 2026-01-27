package factory

import "strings"

// Button callback data constants
const (
	CallbackAddBot     = "add_bot"
	CallbackMyBots     = "my_bots"
	CallbackStats      = "stats"
	CallbackMainMenu   = "main_menu"
	CallbackBotSelect  = "bot_sel"
	CallbackStartBot   = "start_bot"
	CallbackStopBot    = "stop_bot"
	CallbackDeleteBot  = "del_bot"
	CallbackConfirmDel = "conf_del"
	CallbackCancelDel  = "cancel_del"
)

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
