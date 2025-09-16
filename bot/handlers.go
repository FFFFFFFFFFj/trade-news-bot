package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	tb "gopkg.in/telebot.v3"
)

var AdminIDs = map[int64]bool{
	839986298: true,
}

func (b *Bot) IsAdmin(userID int64) bool {
	return AdminIDs[userID]
}

func (b *Bot) HandleMessage(m *tb.Message) error {
	txt := strings.TrimSpace(m.Text)

	switch {
	case txt == "/start":
		subsCount, _ := storage.GetUserSubscriptionCount(b.db, m.Chat.ID)
		if b.IsAdmin(m.Chat.ID) {
			activeUsers, _ := storage.GetActiveUsersCount(b.db)
			msg := fmt.Sprintf("👑 Админ\nID: %d\nАктивных пользователей: %d", m.Chat.ID, activeUsers)
			b.SendMessage(m.Chat.ID, msg)
		} else {
			msg := fmt.Sprintf("👤 Пользователь\nID: %d\nПодписок: %d", m.Chat.ID, subsCount)
			b.SendMessage(m.Chat.ID, msg)
		}

	case txt == "/help":
		b.SendMessage(m.Chat.ID, "Доступные команды:\n/start\n/help\n/latest\n/mysources")

	case txt == "/latest":
		items, _ := storage.GetUnreadNews(b.db, m.Chat.ID, 5)
		if len(items) == 0 {
			b.SendMessage(m.Chat.ID, "Нет новых новостей.")
			return nil
		}
		for _, item := range items {
			_, _ = b.bot.Send(m.Chat, fmt.Sprintf("📰 %s\n%s", item.Title, item.Link))
			_ = storage.MarkNewsAsRead(b.db, m.Chat.ID, item.Link)
		}

	case txt == "/mysources":
		allSources, _ := storage.GetAllSources(b.db)
		userSources, _ := storage.GetUserSources(b.db, m.Chat.ID)

		if len(allSources) == 0 {
			b.SendMessage(m.Chat.ID, "Нет доступных источников.")
			return nil
		}

		userSet := make(map[string]bool)
		for _, s := range userSources {
			userSet[s] = true
		}

		var rows [][]tb.InlineButton
		for _, src := range allSources {
			label := src
			if userSet[src] {
				label = "✅ " + src
			}
			btn := tb.InlineButton{
				Text: label,
				Data: "toggle:" + src,
			}
			rows = append(rows, []tb.InlineButton{btn})
		}

		markup := &tb.ReplyMarkup{InlineKeyboard: rows}
		_, _ = b.bot.Send(m.Chat, "Ваши источники:", markup)

	default:
		log.Printf("Сообщение: %s", txt)
	}

	return nil
}
