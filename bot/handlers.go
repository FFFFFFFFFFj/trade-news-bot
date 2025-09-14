package bot

import (
	"fmt"
	"log"
	"strings"

	//"github.com/FFFFFFFFFFj/trade-news-bot/rss"
	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
)

func (b *Bot) HandleMessage(m *Message) {
	txt := strings.TrimSpace(m.Text)
	switch {
	case txt == "/start":
		b.SendMessage(m.Chat.ID, "👋 Приветствую! Я — ваш бот для получения свежих новостей с инвестиционных сайтов 📈📰.\n\n"+
			"⚡ Чтобы узнать все возможности и как пользоваться ботом, отправьте команду\n"+
			"👉 /help\n\n"+
			"Держите руку на пульсе финансового мира вместе со мной! 🚀💰")

	case txt == "/help":
		helpText := "/start - запустить бота\n" +
			"/latest - последние новости\n" +
			"/help - список команд\n" +
			"/addsource <URL> - добавить источник (админ)\n" +
			"/removesource <URL> - удалить источник (админ)\n" +
			"/listsources - показать все источники (админ)"
		b.SendMessage(m.Chat.ID, helpText)

	case txt == "/latest":
		limit := 5
		items, err := storage.GetLatestNews(b.db, limit)
		if err != nil {
			b.SendMessage(m.Chat.ID, "⚠️ Не удалось загрузить новости из базы данных.")
			log.Printf("GetLatestNews error: %v", err)
			return
		}
		if len(items) == 0 {
			b.SendMessage(m.Chat.ID, "🚫 Сейчас в базе нет свежих новостей.")
			return
		}
		var sb strings.Builder
		count := 0
		for _, item := range items {
			sb.WriteString(fmt.Sprintf("📌 %s\n🕒 %s\n🔗 %s\n\n",
				item.Title,
				item.PubDate,
				item.Link))
			count++
		}
		if count == 0 {
			b.SendMessage(m.Chat.ID, "🚫 Пока нет новых новостей для отправки.")
			return
		}
		b.SendMessage(m.Chat.ID, sb.String())

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
		err := storage.AddSource(b.db, url, 839986298)
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

	default:
		log.Printf("Got message: %s", txt)
	}
}
