package bot

import (
	tb "gopkg.in/telebot.v3"
)

func (b *Bot) ShowSourcesMenu(chatID int64) {
	_, _ = b.bot.Send(
		tb.ChatID(chatID),
		"🔔 Здесь будет меню управления подписками.",
	)
}
