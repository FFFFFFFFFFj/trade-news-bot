package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
)

var AdminIDs = map[int64]bool{
	839986298: true,
}

func (b *Bot) IsAdmin(userID int64) bool {
	return AdminIDs[userID]
}

func (b *Bot) HandleMessage(m *Message) {
	txt := strings.TrimSpace(m.Text)

	// 1) Если у пользователя есть ожидаемое действие (pending) и сообщение не команда — воспринимаем как URL
	if action, ok := b.getPending(m.Chat.ID); ok && !strings.HasPrefix(txt, "/") {
		switch action {
		case "addsource":
			// Добавление источника (только админ)
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
			// Удаление источника (только админ)
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
			// неизвестное состояние — сброс
			b.clearPending(m.Chat.ID)
		}
	}

	// 2) Обработка обычных команд
	switch {
	case txt == "/start":
		// Профиль — пользовательский или админский
		subsCount, err := storage.GetUserSubscriptionCount(b.db, m.Chat.ID)
		if err != nil {
			log.Printf("GetUserSubscriptionCount error for %d: %v", m.Chat.ID, err)
			subsCount = 0
		}
		if b.IsAdmin(m.Chat.ID) {
			activeUsers, err := storage.GetActiveUsersCount(b.db)
			if err != nil {
				log.Printf("GetActiveUsersCount error: %v", err)
				activeUsers = 0
			}
			msg := fmt.Sprintf("👑 Админ профиль\n🆔 Telegram ID: %d\n📊 Активных пользователей: %d\n\nАдмин команды:\n/addsource - добавить источник\n/removesource - удалить источник\n/listsources - показать все источники\n\nПубличные команды:\n/mysources - мои подписки\n/subscribe <URL> - подписаться\n/unsubscribe <URL> - отписаться\n/latest - последние новости\n/help - справка", m.Chat.ID, activeUsers)
			b.SendMessage(m.Chat.ID, msg)
		} else {
			msg := fmt.Sprintf("👤 Ваш профиль\n🆔 Telegram ID: %d\n📌 Подписок: %d\n\nДоступные команды:\n/mysources - мои подписки\n/subscribe <URL> - подписаться\n/unsubscribe <URL> - отписаться\n/latest - последние новости\n/help - справка", m.Chat.ID, subsCount)
			b.SendMessage(m.Chat.ID, msg)
		}
		return

	case txt == "/help":
		helpText := "/start - показать профиль\n" +
			"/latest - последние новости\n" +
			"/help - список команд\n" +
			"/subscribe <URL> - подписаться на источник\n" +
			"/unsubscribe <URL> - отписаться от источника\n" +
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
			err = b.SendMessage(m.Chat.ID, msg)
			if err != nil {
				log.Printf("SendMessage error: %v", err)
				continue
			}
			err = storage.MarkNewsAsRead(b.db, m.Chat.ID, item.Link)
			if err != nil {
				log.Printf("MarkNewsAsRead error: %v", err)
			}
		}

	case strings.HasPrefix(txt, "/addsource"):
		// Для удобства — делаем двухшаговое добавление: сначала команда, бот скажет прислать ссылку
		if !b.IsAdmin(m.Chat.ID) {
			b.SendMessage(m.Chat.ID, "🚫 Команда доступна только администраторам.")
			return
		}
		b.setPending(m.Chat.ID, "addsource")
		b.SendMessage(m.Chat.ID, "Отправьте ссылку на RSS-источник для добавления. Чтобы отменить — отправьте /cancel")

	case strings.HasPrefix(txt, "/removesource"):
		if !b.IsAdmin(m.Chat.ID) {
			b.SendMessage(m.Chat.ID, "🚫 Команда доступна только администраторам.")
			return
		}
		b.setPending(m.Chat.ID, "removesource")
		b.SendMessage(m.Chat.ID, "Отправьте ссылку на RSS-источник для удаления. Чтобы отменить — отправьте /cancel")

	case txt == "/listsources":
		if !b.IsAdmin(m.Chat.ID) {
			b.SendMessage(m.Chat.ID, "🚫 Команда доступна только администраторам.")
			return
		}
		sources, err := storage.GetAllSources(b.db)
		if err != nil {
			b.SendMessage(m.Chat.ID, "Ошибка при получении списка источников.")
			log.Printf("GetAllSources error: %v", err)
			return
		}
		if len(sources) == 0 {
			b.SendMessage(m.Chat.ID, "Список источников пуст.")
			return
		}
		b.SendMessage(m.Chat.ID, "Источники новостей:\n"+strings.Join(sources, "\n"))

	case strings.HasPrefix(txt, "/subscribe"):
		parts := strings.Fields(txt)
		if len(parts) < 2 {
			b.SendMessage(m.Chat.ID, "Использование: /subscribe <URL>")
			return
		}
		url := parts[1]
		err := storage.Subscribe(b.db, m.Chat.ID, url)
		if err != nil {
			b.SendMessage(m.Chat.ID, "Ошибка при подписке: источник не найден или внутренняя ошибка.")
			log.Printf("Subscribe error for %d: %v", m.Chat.ID, err)
			return
		}
		b.SendMessage(m.Chat.ID, "Вы подписаны на источник.")

	case strings.HasPrefix(txt, "/unsubscribe"):
		parts := strings.Fields(txt)
		if len(parts) < 2 {
			b.SendMessage(m.Chat.ID, "Использование: /unsubscribe <URL>")
			return
		}
		url := parts[1]
		err := storage.Unsubscribe(b.db, m.Chat.ID, url)
		if err != nil {
			b.SendMessage(m.Chat.ID, "Ошибка при отписке: источник не найден или внутренняя ошибка.")
			log.Printf("Unsubscribe error for %d: %v", m.Chat.ID, err)
			return
		}
		b.SendMessage(m.Chat.ID, "Вы отписались от источника.")

	case txt == "/mysources":
		urls, err := storage.GetUserSources(b.db, m.Chat.ID)
		if err != nil {
			b.SendMessage(m.Chat.ID, "Ошибка при получении ваших подписок.")
			log.Printf("GetUserSources error for %d: %v", m.Chat.ID, err)
			return
		}
		if len(urls) == 0 {
			b.SendMessage(m.Chat.ID, "У вас пока нет подписок.")
			return
		}
		b.SendMessage(m.Chat.ID, "Ваши подписки:\n"+strings.Join(urls, "\n"))

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
