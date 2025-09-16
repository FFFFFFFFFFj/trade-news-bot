package bot

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"gopkg.in/telebot.v3"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
)

var AdminIDs = map[int64]bool{
	839986298: true, // твой Telegram ID
}

func (b *Bot) IsAdmin(userID int64) bool {
	return AdminIDs[userID]
}

func (b *Bot) HandleMessage(m *telebot.Message) {
	txt := strings.TrimSpace(m.Text)

	// 1) Проверка на pending actions
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
		}
	}

	// 2) Команды
	switch {
	case txt == "/start":
		subsCount, _ := storage.GetUserSubscriptionCount(b.db, m.Chat.ID)

		if b.IsAdmin(m.Chat.ID) {
			activeUsers, _ := storage.GetActiveUsersCount(b.db)
			msg := fmt.Sprintf(
				"👑 Админ профиль\n🆔 Telegram ID: %d\n📊 Активных пользователей: %d\n\nАдмин команды:\n/addsource\n/removesource\n/listsources\n\nПубличные команды:\n/mysources\n/latest\n/help",
				m.Chat.ID, activeUsers,
			)
			b.SendMessage(m.Chat.ID, msg)
		} else {
			msg := fmt.Sprintf(
				"👤 Ваш профиль\n🆔 Telegram ID: %d\n📌 Подписок: %d\n\nДоступные команды:\n/mysources\n/latest\n/help",
				m.Chat.ID, subsCount,
			)
			b.SendMessage(m.Chat.ID, msg)
		}

	case txt == "/help":
		helpText := "/start - показать профиль\n" +
			"/latest - последние новости\n" +
			"/help - список команд\n" +
			"/mysources - мои подписки\n"
		if b.IsAdmin(m.Chat.ID) {
			helpText += "\n(Админ команды)\n" +
				"/addsource - добавить источник\n" +
				"/removesource - удалить источник\n" +
				"/listsources - показать все источники\n"
		}
		b.SendMessage(m.Chat.ID, helpText)

	case txt == "/latest":
		items, _ := storage.GetUnreadNews(b.db, m.Chat.ID, 5)
		if len(items) == 0 {
			b.SendMessage(m.Chat.ID, "🚫 Сейчас нет новых непрочитанных новостей для вас.")
			return
		}
		for _, item := range items {
			msg := fmt.Sprintf("📌 %s\n🕒 %s\n🔗 %s\n\n", item.Title, item.PubDate, item.Link)
			b.SendMessage(m.Chat.ID, msg)
			_ = storage.MarkNewsAsRead(b.db, m.Chat.ID, item.Link)
		}

	case txt == "/mysources":
		allSources, _ := storage.GetAllSources(b.db)
		userSources, _ := storage.GetUserSources(b.db, m.Chat.ID)

		// преобразуем список подписок в map для быстрого поиска
		userSet := make(map[string]bool)
		for _, s := range userSources {
			userSet[s] = true
		}

		var rows [][]telebot.InlineButton
		for _, src := range allSources {
			u, _ := url.Parse(src)
			label := u.Host

			if userSet[src] {
				label = "✅ " + label
			}

			btn := telebot.InlineButton{
				Text:   label,
				Data:   "toggle:" + src,
			}
			rows = append(rows, []telebot.InlineButton{btn})
		}

		markup := &telebot.ReplyMarkup{InlineKeyboard: rows}
		b.bot.Send(m.Chat, "Ваши источники:", markup)

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
