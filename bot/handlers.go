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

	// используем if-else вместо switch, чтобы корректно работать с HasPrefix
	if txt == "/start" {
		subsCount, _ := storage.GetUserSubscriptionCount(b.db, m.Chat.ID)
		if b.IsAdmin(m.Chat.ID) {
			activeUsers, _ := storage.GetActiveUsersCount(b.db)
			msg := fmt.Sprintf("👑 Админ\nID: %d\nАктивных пользователей: %d\nВсего источников: %d",
				m.Chat.ID, activeUsers, len(storage.MustGetAllSources(b.db)))
			b.SendMessage(m.Chat.ID, msg)
		} else {
			msg := fmt.Sprintf("👤 Пользователь\nID: %d\nПодписок: %d", m.Chat.ID, subsCount)
			b.SendMessage(m.Chat.ID, msg)
		}

	} else if txt == "/help" {
		b.SendMessage(m.Chat.ID, "Доступные команды:\n" +
			"/start – информация о вас\n" +
			"/help – список команд\n" +
			"/latest – новости за сегодня с пагинацией\n" +
			"/mysources – управление подписками\n" +
			"/autopost – настройка авторассылки (0–6 раз в день, время по Москве)\n" +
			"Можно также указать вручную: /autopost 10:30 15:45\n")

	} else if strings.HasPrefix(txt, "/autopost ") {
		parts := strings.Fields(txt)[1:]
		var validTimes []string
		for _, p := range parts {
			if len(p) == 5 && p[2] == ':' {
				validTimes = append(validTimes, p)
			}
		}
		if len(validTimes) > 6 {
			b.SendMessage(m.Chat.ID, "⚠️ Можно максимум 6 времён")
		} else if len(validTimes) == 0 {
			b.SendMessage(m.Chat.ID, "⚠️ Неверный формат времени. Используйте ЧЧ:ММ, например /autopost 09:00 15:30")
		} else {
			_ = storage.SetUserAutopost(b.db, m.Chat.ID, validTimes)
			b.SendMessage(m.Chat.ID, "✅ Время авторассылки обновлено: "+strings.Join(validTimes, ", "))
		}

	} else if txt == "/autopost" {
		b.ShowAutopostMenu(m.Chat.ID)

	} else if txt == "/latest" {
		b.SendMessage(m.Chat.ID, "⏳ Загружаю сегодняшние новости...")
		b.latestPage[m.Chat.ID] = 1
		b.ShowLatestNews(m.Chat.ID, nil)

	} else if txt == "/mysources" {
		b.ShowSourcesMenu(m.Chat.ID)

	} else {
		log.Printf("Сообщение: %s", txt)
	}
}
