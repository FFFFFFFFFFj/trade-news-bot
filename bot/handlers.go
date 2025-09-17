package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	tb "gopkg.in/telebot.v3"
)

var AdminIDs = map[int64]bool{
	839986298: true, // твой ID
}

func (b *Bot) IsAdmin(userID int64) bool {
	return AdminIDs[userID]
}

func (b *Bot) HandleMessage(m *tb.Message) {
	txt := strings.TrimSpace(m.Text)

	switch {
	case txt == "/start":
		subsCount, _ := storage.GetUserSubscriptionCount(b.db, m.Chat.ID)
		if b.IsAdmin(m.Chat.ID) {
			activeUsers, _ := storage.GetActiveUsersCount(b.db)
			msg := fmt.Sprintf(
				"👑 Админ\nID: %d\nАктивных пользователей: %d\nВсего источников: %d",
				m.Chat.ID, activeUsers, len(storage.MustGetAllSources(b.db)),
			)
			b.SendMessage(m.Chat.ID, msg)
		} else {
			msg := fmt.Sprintf("👤 Пользователь\nID: %d\nПодписок: %d", m.Chat.ID, subsCount)
			b.SendMessage(m.Chat.ID, msg)
		}

	case txt == "/help":
		b.SendMessage(m.Chat.ID, "Доступные команды:\n/start\n/help\n/latest\n/mysources")

	case txt == "/latest":
		b.latestPage[m.Chat.ID] = 1
		b.ShowLatestNews(m.Chat.ID, nil)

	case txt == "/mysources":
		b.ShowSourcesMenu(m.Chat.ID)

	default:
		log.Printf("Сообщение: %s", txt)
	}
}
