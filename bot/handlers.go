package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	tb "gopkg.in/telebot.v3"
)

func (b *Bot) HandleMessage(m *tb.Message) {
	userID := m.Chat.ID
	_, _ = b.db.Exec(`INSERT INTO users (id) VALUES ($1) ON CONFLICT DO NOTHING`, userID)

	txt := strings.TrimSpace(m.Text)

	// Проверяем, ожидает ли админ ввод URL или рассылки
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
			b.pending[userID] = ""
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
			b.pending[userID] = ""
			return

		case "broadcast":
			if txt == "" {
				b.SendMessage(userID, "⚠️ Сообщение пустое")
				return
			}
			count := b.BroadcastMessageToAll(txt)
			b.SendMessage(userID, fmt.Sprintf("✅ Сообщение разослано %d пользователям", count))
			b.pending[userID] = ""
			return
		}
	}

	switch {
	case txt == "/start":
		if b.IsAdmin(userID) {
			usersCount, _ := storage.GetUsersCount(b.db)
			activeUsers, _ := storage.GetActiveUsersCount(b.db)
			autopostUsers, _ := storage.GetAutopostUsersCount(b.db)
			msg := fmt.Sprintf("👑 Админ\nID: %d\nВсего пользователей: %d\nПодписанных: %d\nС автопостом: %d\nВсего источников: %d",
				userID, usersCount, activeUsers, autopostUsers, len(storage.MustGetAllSources(b.db)))
			b.SendMessage(userID, msg)
		} else {
			subsCount, _ := storage.GetUserSubscriptionCount(b.db, userID)
			msg := fmt.Sprintf("👤 Пользователь\nID: %d\nПодписок: %d", userID, subsCount)
			b.SendMessage(userID, msg)
		}

	case txt == "/help":
		helpMsg := "Доступные команды:\n" +
			"/start – информация о вас\n" +
			"/help – список команд\n" +
			"/latest – новости за сегодня\n" +
			"/mysources – управление подписками\n" +
			"/autopost – настройка авторассылки\n"
		if b.IsAdmin(userID) {
			helpMsg += "\nАдмин-команды:\n" +
				"/addsource – добавить источник\n" +
				"/removesource – удалить источник\n" +
				"/listsources – список источников\n" +
				"/broadcast – рассылка всем\n"
		}
		b.SendMessage(userID, helpMsg)

	case txt == "/autopost":
		b.ShowAutopostMenu(userID)

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

	case txt == "/latest":
		b.latestPage[userID] = 1
		b.ShowLatestNews(userID, nil)

	case txt == "/mysources":
		b.ShowSourcesMenu(userID)

	case txt == "/addsource":
		if !b.IsAdmin(userID) {
			return
		}
		b.pending[userID] = "addsource"
		b.SendMessage(userID, "Введите URL нового источника:")

	case txt == "/removesource":
		if !b.IsAdmin(userID) {
			return
		}
		b.pending[userID] = "removesource"
		b.SendMessage(userID, "Введите URL источника для удаления:")

	case txt == "/broadcast":
		if !b.IsAdmin(userID) {
			return
		}
		b.pending[userID] = "broadcast"
		b.SendMessage(userID, "Введите текст для рассылки всем пользователям:")

	case txt == "/listsources":
		if !b.IsAdmin(userID) {
			return
		}
		sources := storage.MustGetAllSources(b.db)
		if len(sources) == 0 {
			b.SendMessage(userID, "Источники не найдены")
		} else {
			msg := "Список источников:\n" + strings.Join(sources, "\n")
			b.SendMessage(userID, msg)
		}

	default:
		log.Printf("Сообщение: %s", txt)
	}
}

// BroadcastMessageToAll разсылает текстовое сообщение всем пользователям
func (b *Bot) BroadcastMessageToAll(msg string) int {
	rows, err := b.db.Query(`SELECT id FROM users`)
	if err != nil {
		log.Printf("Ошибка получения пользователей: %v", err)
		return 0
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
	return count
}
