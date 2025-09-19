package bot

import (
	"fmt"
	tb "gopkg.in/telebot.v3"
)

func (b *Bot) ShowAutopostMenu(chatID int64) {
	menu := &tb.ReplyMarkup{}
	var rows [][]tb.InlineButton

	for hour := 0; hour < 24; hour++ {
		// создаём кнопку через меню
		btn := menu.Data(fmt.Sprintf("%02d:00", hour), fmt.Sprintf("ap_%02d00", hour))
		// преобразуем tb.Btn в tb.InlineButton
		rows = append(rows, []tb.InlineButton{
			tb.InlineButton{
				Unique: btn.Unique,
				Text:   btn.Text,
			},
		})
	}
	menu.InlineKeyboard = rows

	_, _ = b.bot.Send(
		tb.ChatID(chatID),
		"Выберите время авторассылки (по Москве):",
		menu,
	)
}
