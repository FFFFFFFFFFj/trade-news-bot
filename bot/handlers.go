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
	switch {
	case txt == "/start":
		b.SendMessage(m.Chat.ID, "👋 Приветствую! Я — ваш бот для получения свежих новостей 📈📰.\n\n"+
			"⚡ Чтобы узнать все возможности и как пользоваться ботом, отправьте команду\n"+
			"👉 /help\n\n"+
			"Держите руку на пульсе финансового мира вместе со мной! 🚀💰")

	case txt == "/help":
		helpText := "/start - запустить бота\n" +
			"/latest - последние новости (некоторые могут быть помечены прочитанными)\n" +
			"/subscribe <URL> - подписаться на источник\n" +
			"/unsubscribe <URL> - отписаться от источника\n" +
			"/mysources - показать мои подписки\n"

		if b.IsAdmin(m.Chat.ID) {
			helpText += "\n(Админ команды)\n" +
				"/addsource <URL>\n" +
				"/removesource <URL>\n" +
				"/listsources"
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
		if !b.IsAdmin(m.Chat.ID) {
			b.SendMessage(m.Chat.ID, "🚫 Команда доступна только администраторам.")
			return
		}
		parts := strings.Fields(txt)
		if len(parts) < 2 {
			b.SendMessage(m.Chat.ID, "Использование: /addsource <URL>")
			return
		}
		url := parts[1]
		err := storage.AddSource(b.db, url, m.Chat.ID)
		if err != nil {
			b.SendMessage(m.Chat.ID, "Ошибка при добавлении источника.")
			log.Printf("AddSource error: %v", err)
			return
		}
		b.SendMessage(m.Chat.ID, "Источник успешно добавлен.")

	case strings.HasPrefix(txt, "/removesource"):
		if !b.IsAdmin(m.Chat.ID) {
			b.SendMessage(m.Chat.ID, "🚫 Команда доступна только администраторам.")
			return
		}
		parts := strings.Fields(txt)
		if len(parts) < 2 {
			b.SendMessage(m.Chat.ID, "Использование: /removesource <URL>")
			return
		}
		url := parts[1]
		err := storage.RemoveSource(b.db, url)
		if err != nil {
			b.SendMessage(m.Chat.ID, "Ошибка при удалении источника.")
			log.Printf("RemoveSource error: %v", err)
			return
		}
		b.SendMessage(m.Chat.ID, "Источник успешно удалён.")

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

	// --- подписки для пользователей ---
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

	default:
		log.Printf("Got message: %s", txt)
	}
}
