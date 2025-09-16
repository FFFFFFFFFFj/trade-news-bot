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

	// 1️⃣ Pending actions (для /addsource и /removesource)
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
				log.Printf("AddSource error: %v", err)
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
				log.Printf("RemoveSource error: %v", err)
			} else {
				b.SendMessage(m.Chat.ID, "Источник успешно удалён.")
			}
			b.clearPending(m.Chat.ID)
			return
		default:
			b.clearPending(m.Chat.ID)
		}
	}

	// 2️⃣ Основные команды
	switch {
	case txt == "/start":
		// Добавляем пользователя в базу, если первый раз
		err := storage.AddUserIfNotExists(b.db, m.Chat.ID)
		if err != nil {
			log.Printf("AddUserIfNotExists error for %d: %v", m.Chat.ID, err)
		}

		subsCount, _ := storage.GetUserSubscriptionCount(b.db, m.Chat.ID)

		if b.IsAdmin(m.Chat.ID) {
			activeUsers, _ := storage.GetActiveUsersCount(b.db)
			totalUsers, _ := storage.GetTotalUsersCount(b.db)

			msg := fmt.Sprintf(
				"👑 Админ профиль\n🆔 Telegram ID: %d\n📊 Активных пользователей: %d\n🌐 Всего пользователей, нажавших /start: %d\n\n"+
					"Админ команды:\n/addsource - добавить источник\n/removesource - удалить источник\n/listsources - показать все источники\n\n"+
					"Публичные команды:\n/mysources - мои подписки\n/subscribe - подписаться\n/unsubscribe - отписаться\n/latest - последние новости\n/help - справка",
				m.Chat.ID, activeUsers, totalUsers)
			b.SendMessage(m.Chat.ID, msg)
		} else {
			msg := fmt.Sprintf(
				"👤 Ваш профиль\n🆔 Telegram ID: %d\n📌 Подписок: %d\n\nДоступные команды:\n/mysources - мои подписки\n/subscribe - подписаться\n/unsubscribe - отписаться\n/latest - последние новости\n/help - справка",
				m.Chat.ID, subsCount)
			b.SendMessage(m.Chat.ID, msg)
		}
		return

	case txt == "/help":
		helpText := "/start - показать профиль\n" +
			"/latest - последние новости\n" +
			"/help - список команд\n" +
			"/mysources - показать мои подписки\n"
		if b.IsAdmin(m.Chat.ID) {
			helpText += "\n(Админ команды)\n" +
				"/addsource - добавить источник (в два шага)\n" +
				"/removesource - удалить источник (в два шага)\n" +
				"/listsources - показать все источники\n"
		}
		b.SendMessage(m.Chat.ID, helpText)

	case txt == "/latest":
		limit := 5
		items, err := storage.GetUnreadNews(b.db, m.Chat.ID, limit)
		if err != nil {
			b.SendMessage(m.Chat.ID, "⚠️ Не удалось загрузить новости из базы данных.")
			log.Printf("GetUnreadNews error: %v", err)
			return
		}
		if len(items) == 0 {
			b.SendMessage(m.Chat.ID, "🚫 Сейчас нет новых непрочитанных новостей для вас.")
			return
		}
		for _, item := range items {
			msg := fmt.Sprintf("📌 %s\n🕒 %s\n🔗 %s\n\n", item.Title, item.PubDate, item.Link)
			_ = b.SendMessage(m.Chat.ID, msg)
			_ = storage.MarkNewsAsRead(b.db, m.Chat.ID, item.Link)
		}

	case txt == "/mysources":
		// Получаем все источники и подписки пользователя
		allSources, err := storage.GetAllSources(b.db)
		if err != nil {
			b.SendMessage(m.Chat.ID, "Ошибка при получении списка источников.")
			log.Printf("GetAllSources error: %v", err)
			return
		}
		userSources, err := storage.GetUserSources(b.db, m.Chat.ID)
		if err != nil {
			b.SendMessage(m.Chat.ID, "Ошибка при получении ваших подписок.")
			log.Printf("GetUserSources error: %v", err)
			return
		}

		// Строим инлайн кнопки
		var buttons [][]tb.InlineButton
		for _, src := range allSources {
			displayName := src
			// Выделяем домен для кнопки
			if u, err := url.Parse(src); err == nil {
				displayName = u.Host
			}
			prefix := ""
			if contains(userSources, src) {
				prefix = "✅ "
			}
			btn := tb.InlineButton{
				Unique: "toggle_sub_" + displayName,
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

// Вспомогательная функция
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
