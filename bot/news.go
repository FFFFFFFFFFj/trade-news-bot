package bot

import (
	"fmt"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	tb "gopkg.in/telebot.v3"
)

func (b *Bot) ShowLatestNews(chatID int64, c tb.Context) {
	page := b.latestPage[chatID]
	if page < 1 {
		page = 1
	}
	pageSize := 4

	// Используем существующую функцию из storage
	news, _ := storage.GetLatestNewsPageForUser(b.db, chatID, page, pageSize)
	if len(news) == 0 {
		b.SendMessage(chatID, "Сегодня новостей нет.")
		return
	}

	text := "📰 Новости за сегодня:\n\n"
	for _, n := range news {
		text += fmt.Sprintf("• <b>%s</b>\n%s\n\n", n.Title, n.Link)
	}

	// считаем страницы
	totalCount, _ := storage.GetTodayNewsCountForUser(b.db, chatID)
	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	// добавляем счётчик
	text += fmt.Sprintf("📄 Страница %d/%d", page, totalPages)

	btns := [][]tb.InlineButton{}
	row := []tb.InlineButton{}
	if page > 1 {
		row = append(row, b.btnFirst, b.btnPrev)
	}
	if page < totalPages {
		row = append(row, b.btnNext, b.btnLast)
	}
	if len(row) > 0 {
		btns = append(btns, row)
	}

	_, _ = b.bot.Send(
		tb.ChatID(chatID),
		text,
		&tb.SendOptions{ParseMode: tb.ModeHTML, ReplyMarkup: &tb.ReplyMarkup{InlineKeyboard: btns}},
	)
}
