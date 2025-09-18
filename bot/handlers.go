package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	tb "gopkg.in/telebot.v3"
)

func (b *Bot) HandleMessage(m *tb.Message) {
	_, _ = b.db.Exec(`INSERT INTO users (id) VALUES ($1) ON CONFLICT DO NOTHING`, m.Chat.ID)
	txt := strings.TrimSpace(m.Text)

	if txt == "/start" {
		if b.IsAdmin(m.Chat.ID) {
			usersCount, _ := storage.GetUsersCount(b.db)
			activeUsers, _ := storage.GetActiveUsersCount(b.db)
			autopostUsers, _ := storage.GetAutopostUsersCount(b.db)
			msg := fmt.Sprintf("👑 Админ\nID: %d\nВсего пользователей: %d\nПодписанных: %d\nС автопостом: %d\nВсего источников: %d",
				m.Chat.ID, usersCount, activeUsers, autopostUsers, len(storage.MustGetAllSources(b.db)))
			b.SendMessage(m.Chat.ID, msg)
		} else {
			subsCount, _ := storage.GetUserSubscriptionCount(b.db, m.Chat.ID)
			msg := fmt.Sprintf("👤 Пользователь\nID: %d\nПодписок: %d", m.Chat.ID, subsCount)
			b.SendMessage(m.Chat.ID, msg)
		}

	} else if txt == "/help" {
		b.SendMessage(m.Chat.ID, "Доступные команды:\n" +
			"/start – информация о вас\n" +
			"/help – список команд\n" +
			"/latest – новости за сегодня\n" +
			"/mysources – управление подписками\n" +
			"/autopost – настройка авторассылки\n")

	} else if strings.HasPrefix(txt, "/autopost ") {
		parts := strings.Fields(txt)[1:]
		var validTimes []string
		for _, p := range parts {
			if len(p) == 5 && p[2] == ':' {
				validTimes = append(validTimes, p)
			}
		}
		if len(validTimes) > 6 {
			b.SendMessage(m.Chat.ID, "⚠️ Максимум 6 времён")
		} else if len(validTimes) == 0 {
			b.SendMessage(m.Chat.ID, "⚠️ Неверный формат времени")
		} else {
			_ = storage.SetUserAutopost(b.db, m.Chat.ID, validTimes)
			b.SendMessage(m.Chat.ID, "✅ Время авторассылки обновлено: "+strings.Join(validTimes, ", "))
		}

	} else if txt == "/autopost" {
		b.ShowAutopostMenu(m.Chat.ID)

	} else if txt == "/latest" {
		b.latestPage[m.Chat.ID] = 1
		b.ShowLatestNews(m.Chat.ID, nil)

	} else if txt == "/mysources" {
		b.ShowSourcesMenu(m.Chat.ID)

	} else {
		log.Printf("Сообщение: %s", txt)
	}
}
