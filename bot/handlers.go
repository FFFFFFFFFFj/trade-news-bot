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
	switch txt {
	case "/start":
		b.SendMessage(m.Chat.ID, "👋 Приветствую! Я — ваш бот для получения свежих новостей с инвестиционных сайтов 📈📰.\n\n"+
			"⚡ Чтобы узнать все возможности и как пользоваться ботом, отправьте команду\n"+
			"👉 /help\n\n"+
			"Держите руку на пульсе финансового мира вместе со мной! 🚀💰")

	case "/help":
		helpText := "/start - запустить бота\n" +
			"/latest - последние новости\n" +
			"/help - список команд"
		b.SendMessage(m.Chat.ID, helpText)

	case "/latest":
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
			if b.Sent[item.Link] {
				continue // уже отправили
			}
			sb.WriteString(fmt.Sprintf("📌 %s\n🕒 %s\n🔗 %s\n\n",
				item.Title,
				item.PubDate,
				item.Link))
			b.Sent[item.Link] = true
			count++
		}
		if count == 0 {
			b.SendMessage(m.Chat.ID, "🚫 Пока нет новых новостей для отправки.")
            return
		}
		b.SendMessage(m.Chat.ID, sb.String())

	default:
		log.Printf("Got message: %s", txt)
	}
}
