package bot

import (
	"fmt"
	"log"
	"net/url"
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

func (b *Bot) HandleMessage(m *Message) {
	txt := strings.TrimSpace(m.Text)

	// Pending actions
	if action, ok := b.getPending(m.Chat.ID); ok && !strings.HasPrefix(txt, "/") {
		switch action {
		case "addsource":
			if !b.IsAdmin(m.Chat.ID) {
				b.SendMessage(m.Chat.ID, "🚫 Команда доступна только администраторам.")
				b.clearPending(m.Chat.ID)
				return
			}
			err := storage.AddSource(b.db, txt, m.Chat.ID)
			if err != nil {
				b.SendMessage(m.Chat.ID, "Ошибка при добавлении источника.")
			} else {
				b.SendMessage(m.Chat.ID, "Источник успешно добавлен.")
			}
			b.clearPending(m.Chat.ID)
			return

		case "removesource":
			if !b.IsAdmin(m.Chat.ID) {
				b.SendMessage(m.Chat.ID, "🚫 Команда доступна только администраторам.")
				b.clearPending(m.Chat.ID)
				return
			}
			err := storage.RemoveSource(b.db, txt)
			if err != nil {
				b.SendMessage(m.Chat.ID, "Ошибка при удалении источника.")
			} else {
				b.SendMessage(m.Chat.ID, "Источник успешно удалён.")
			}
			b.clearPending(m.Chat.ID)
			return

		default:
			b.clearPending(m.Chat.ID)
		}
	}

	switch {
	case txt == "/start":
		_ = storage.AddUserIfNotExists(b.db, m.Chat.ID)

		subsCount, _ := storage.GetUserSubscriptionCount(b.db, m.Chat.ID)

		if b.IsAdmin(m.Chat.ID) {
			activeUsers, _ := storage.GetActiveUsersCount(b.db)
			totalUsers, _ := storage.GetTotalUsersCount(b.db)

			msg := fmt.Sprintf(
				"👑 Админ профиль\n🆔 Telegram ID: %d\n📊 Активных пользователей: %d\n🌐 Всего пользователей, нажавших /start: %d\n\n"+
					"Админ команды:\n/addsource\n/removesource\n/listsources\n\n"+
					"Публичные команды:\n/mysources\n/subscribe\n/unsubscribe\n/latest\n/help",
				m.Chat.ID, activeUsers, totalUsers)
			b.SendMessage(m.Chat.ID, msg)
		} else {
			msg := fmt.Sprintf(
				"👤 Ваш профиль\n🆔 Telegram ID: %d\n📌 Подписок: %d\n\nДоступные команды:\n/mysources\n/subscribe\n/unsubscribe\n/latest\n/help",
				m.Chat.ID, subsCount)
			b.SendMessage(m.Chat.ID, msg)
		}
		return

	case txt == "/help":
		helpText := "/start - показать профиль\n/latest - последние новости\n/help - список команд\n/mysources - мои подписки\n"
		if b.IsAdmin(m.Chat.ID) {
			helpText += "/addsource - добавить источник\n/removesource - удалить источник\n/listsources - показать все источники\n"
		}
		b.SendMessage(m.Chat.ID, helpText)

	case txt == "/latest":
		limit := 5
		items, err := storage.GetUnreadNews(b.db, m.Chat.ID, limit)
		if err != nil {
			b.SendMessage(m.Chat.ID, "⚠️ Не удалось загрузить новости из базы данных.")
			return
		}
		if len(items) == 0 {
			b.SendMessage(m.Chat.ID, "🚫 Сейчас нет новых непрочитанных новостей.")
			return
		}
		for _, item := range items {
			msg := fmt.Sprintf("📌 %s\n🕒 %s\n🔗 %s\n\n", item.Title, item.PubDate, item.Link)
			b.SendMessage(m.Chat.ID, msg)
			storage.MarkNewsAsRead(b.db, m.Chat.ID, item.Link)
		}

	case txt == "/mysources":
		allSources, err := storage.GetAllSources(b.db)
		if err != nil {
			b.SendMessage(m.Chat.ID, "Ошибка при получении списка источников.")
			return
		}
		userSources, err := storage.GetUserSources(b.db, m.Chat.ID)
		if err != nil {
			b.SendMessage(m.Chat.ID, "Ошибка при получении ваших подписок.")
			return
		}

		var buttons [][]tb.InlineButton
		for _, src := range allSources {
			displayName := src
			if u, err := url.Parse(src); err == nil {
				displayName = u.Host
			}
			prefix := ""
			if contains(userSources, src) {
				prefix = "✅ "
			}
			btn := tb.InlineButton{
				Unique: "toggle_" + displayName,
				Text:   prefix + displayName,
				Data:   src,
			}
			buttons = append(buttons, []tb.InlineButton{btn})
		}
		b.SendInlineButtons(m.Chat.ID, "Ваши подписки:", buttons)

	case txt == "/cancel":
		if _, ok := b.getPending(m.Chat.ID); ok {
			b.clearPending(m.Chat.ID)
			b.SendMessage(m.Chat.ID, "Операция отменена.")
		} else {
			b.SendMessage(m.Chat.ID, "Нечего отменять.")
		}

	default:
		log.Printf("Got message: %s", txt)
	}
}
