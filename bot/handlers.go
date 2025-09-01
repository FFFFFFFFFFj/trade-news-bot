package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/FFFFFFFFFFj/trade-news-bot/rss"
)

func (b *Bot) HandleMessage(m *Message) {
	txt := strings.TrimSpace(m.Text)
	switch txt {
	case "/start":
		b.SendMessage(m.Chat.ID, "👋 Приветствую! Я — ваш бот для получения свежих новостей с инвестиционных сайтов 📈📰.\n\n" +
					  "⚡ Чтобы узнать все возможности и как пользоваться ботом, отправьте команду\n" +
					  "👉 /help\n\n" +
					  "Держите руку на пульсе финансового мира вместе со мной! 🚀💰")
	
	case "/help":
		helpText := "/start - запустить бота\n" +
					"/latest - последние новости\n" +
					"/help - список команд"
		b.SendMessage(m.Chat.ID, helpText) 
		
	case "/latest":
		items, err := rss.Fetch("https://www.investing.com/rss/news.rss")
		if err != nil {
			b.SendMessage(m.Chat.ID, "⚠️ Ошибка загрузки новостей!" +
						 "Возникла проблема с получением свежих данных с источников. 🛑" +
						 "Пожалуйста, попробуйте позже или отправьте команду /help для дополнительной" +
						 "информации. 🙏")
			return
		}

		limit := 3
		if len(items) < limit {
			limit = len(items)
		}

		var sb strings.Builder
		for i := 0; i < limit; i++ {
			if b.Sent[items[i].Link] {
				continue // already sent
			}
			sb.WriteString(fmt.Sprintf("📌 %s\n🕒 %s\n🔗 %s\n\n", 
				items[i].Title, 
				items[i].PubDate,
				items[i].Link,))
			b.Sent[items[i].Link] = true // mark as sent
		}

		if sb.Len() == 0 {
			b.SendMessage(m.Chat.ID, "🚫 Упс! Пока на выбранных ресурсах нет свежих новостей 📉🕰️.\n\n" +
						 "Пожалуйста, попробуйте позже или отправьте команду 👉 /help для получения информации о возможностях бота.\n\n" +
						 "Спасибо, что остаетесь с нами! 💼✨")
			return
		}

		b.SendMessage(m.Chat.ID, sb.String())

		
	default:
		log.Printf("Got message: %s", txt)
	}
}
