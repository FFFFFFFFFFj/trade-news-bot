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
	userID := m.Chat.ID

	// Проверяем, ожидает ли админ ввод URL или текста для рассылки
	if mode, ok := b.pending[userID]; ok && b.IsAdmin(userID) {
		switch mode {
		case "addsource":
			if txt == "" {
				b.SendMessage(userID, "⚠️ URL пустой")
				return
			}
			if err := storage.AddSource(b.db, txt); err != nil {
				b.SendMessage(userID, "❌ Ошибка добавления источника")
			} else {
				b.SendMessage(userID, "✅ Источник добавлен: "+txt)
			}
			b.pending[userID] = "" // сброс режима
			return

		case "removesource":
			if txt == "" {
				b.SendMessage(userID, "⚠️ URL пустой")
				return
			}
			if err := storage.RemoveSource(b.db, txt); err != nil {
				b.SendMessage(userID, "❌ Ошибка удаления источника")
			} else {
				b.SendMessage(userID, "✅ Источник удалён: "+txt)
			}
			b.pending[userID] = "" // сброс режима
			return

		case "broadcast":
			if txt == "" {
				b.SendMessage(userID, "⚠️ Сообщение пустое")
				return
			}
			b.BroadcastMessage(txt)
			b.SendMessage(userID, "✅ Сообщение разослано всем пользователям")
			b.pending[userID] = "" // сброс режима
			return
		}
	}

	// Обработка обычных команд
	switch {
	case txt == "/start":
		if b.IsAdmin(userID) {
			usersCount, _ := storage.GetUsersCount(b.db)
			activeUsers, _ := storage.GetActiveUsersCount(b.db)
			autopostUsers, _ := storage.GetAutopostUsersCount(b.db)
			msg := fmt.Sprintf(
				"👑 Админ\nID: %d\nВсего пользователей: %d\nПодписанных: %d\nС автопостом: %d\nВсего источников: %d",
				userID, usersCount, activeUsers, autopostUsers, len(storage.MustGetAllSources(b.db)),
			)
			b.SendMessage(userID, msg)
		} else {
			subsCount, _ := storage.GetUserSubscriptionCount(b.db, userID)
			msg := fmt.Sprintf("👤 Пользователь\nID: %d\nПодписок: %d", userID, subsCount)
			b.SendMessage(userID, msg)
		}

	case txt == "/help":
		b.SendMessage(userID, "Доступные команды:\n"+
			"/start – информация о вас\n"+
			"/help – список команд\n"+
			"/latest – новости за сегодня\n"+
			"/mysources – управление подписками\n"+
			"/autopost – настройка авторассылки\n")

	case strings.HasPrefix(txt, "/autopost "):
		parts := strings.Fields(txt)[1:]
		var validTimes []string
		for _, p := range parts {
			if len(p) == 5 && p[2] == ':' {
				validTimes = append(validTimes, p)
			}
		}
		if len(validTimes) > 6 {
			b.SendMessage(userID, "⚠️ Максимум 6 времён")
		} else if len(validTimes) == 0 {
			b.SendMessage(userID, "⚠️ Неверный формат времени")
		} else {
			_ = storage.SetUserAutopost(b.db, userID, validTimes)
			b.SendMessage(userID, "✅ Время авторассылки обновлено: "+strings.Join(validTimes, ", "))
		}

	case txt == "/autopost":
		b.ShowAutopostMenu(userID)

	case txt == "/latest":
		b.latestPage[userID] = 1
		b.ShowLatestNews(userID, nil)

	case txt == "/mysources":
		b.ShowSourcesMenu(userID)

	default:
		log.Printf("Сообщение: %s", txt)
	}
}

// BroadcastMessage разсылает текст всем пользователям
func (b *Bot) BroadcastMessage(msg string) {
	rows, err := b.db.Query(`SELECT id FROM users`)
	if err != nil {
		log.Printf("Ошибка получения пользователей: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err == nil {
			b.SendMessage(uid, msg)
			count++
		}
	}
	log.Printf("Сообщение разослано %d пользователям", count)
}
