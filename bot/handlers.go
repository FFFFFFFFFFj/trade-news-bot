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

	// Проверяем, ожидает ли админ ввод URL или сообщение для рассылки
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
			b.pending[userID] = txt
			b.HandleAdminBroadcast(&tb.Callback{Sender: m.Sender}) // вызов рассылки
			return
		}
	}

	switch {
	case txt == "/start":
		if b.IsAdmin(userID) {
			b.ShowAdminMenu(userID) // Показываем админские кнопки
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

// ---------------------- Админское меню с кнопками ----------------------

func (b *Bot) ShowAdminMenu(chatID int64) {
	if !b.IsAdmin(chatID) {
		return
	}

	markup := &tb.ReplyMarkup{}
	btnAdd := tb.InlineButton{
		Unique: "admin_addsource",
		Text:   "➕ Добавить источник",
	}
	btnRemove := tb.InlineButton{
		Unique: "admin_removesource",
		Text:   "➖ Удалить источник",
	}
	btnBroadcast := tb.InlineButton{
		Unique: "admin_broadcast",
		Text:   "📢 Рассылка всем",
	}

	markup.InlineKeyboard = [][]tb.InlineButton{
		{btnAdd, btnRemove},
		{btnBroadcast},
	}

	_, _ = b.bot.Send(tb.ChatID(chatID), "👑 Админ-меню:", markup)

	// Привязка кнопок к режимам pending
	b.bot.Handle(&btnAdd, func(c tb.Context) error {
		b.pending[chatID] = "addsource"
		return c.Respond(&tb.CallbackResponse{Text: "Введите URL для добавления источника"})
	})
	b.bot.Handle(&btnRemove, func(c tb.Context) error {
		b.pending[chatID] = "removesource"
		return c.Respond(&tb.CallbackResponse{Text: "Введите URL для удаления источника"})
	})
	b.bot.Handle(&btnBroadcast, func(c tb.Context) error {
		b.pending[chatID] = "broadcast"
		return c.Respond(&tb.CallbackResponse{Text: "Введите сообщение для рассылки"})
	})
}
