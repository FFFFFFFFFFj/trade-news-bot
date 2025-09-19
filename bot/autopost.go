package bot

import (
	"fmt"
	tb "gopkg.in/telebot.v3"
)

func (b *Bot) ShowAutopostMenu(chatID int64) {
	menu := &tb.ReplyMarkup{}
	var rows [][]tb.Btn

	// создаём кнопки выбора времени
	for hour := 0; hour < 24; hour++ {
		btn := menu.Data(fmt.Sprintf("%02d:00", hour), fmt.Sprintf("ap_%02d00", hour))
		rows = append(rows, []tb.Btn{btn})
	}
	menu.InlineKeyboard = rows

	_, _ = b.bot.Send(
		tb.ChatID(chatID),
		"Выберите время авторассылки (по Москве):",
		menu,
	)
}
