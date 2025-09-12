package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/FFFFFFFFFFj/trade-news-bot/rss"
	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
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
		sources := []string{
			"https://www.finmarket.ru/rss/main.asp",
    		"https://www.finmarket.ru/rss/ecworld.asp",
    		"https://www.finmarket.ru/rss/finances.asp",
    		"https://www.bfm.ru/rss/news.xml",
    		"https://rssexport.rbc.ru/rbcnews/economics/full.rss",
    		"https://rssexport.rbc.ru/rbcnews/finance/full.rss",
    		"https://rssexport.rbc.ru/rbcnews/business/full.rss",
    		"https://www.interfax.ru/rss.asp",
    		"https://tass.ru/rss/v2/economy.xml",
    		"https://ru.investing.com/rss/news.rss",
    		"https://ru.investing.com/rss/forex.rss",
    		"https://ru.investing.com/rss/cryptocurrency.rss",
		}

		
		items, err := rss.FetchAll(sources)
		if err != nil {
			b.SendMessage(m.Chat.ID, "⚠️ Ошибка загрузки новостей!" +
						 "Возникла проблема с получением свежих данных с источников. 🛑" +
						 "Пожалуйста, попробуйте позже или отправьте команду /help для дополнительной" +
						 "информации. 🙏")
			return
		}

		//save news in base
		for _, item := range items {
			err := storage.SaveNews(db, item, sourceURL)
			if err != nil {
				log.Printf("SaveNews error: %v", err)
			}
		}

		limit := 5
		if len(items) < limit {
			limit = len(items)
		}

		var sb strings.Builder
		count := 0
		for i := 0; i < limit; i++ {
			if b.Sent[items[i].Link] {
				continue // already sent
			}
			sb.WriteString(fmt.Sprintf("📌 %s\n🕒 %s\n🔗 %s\n\n", 
				items[i].Title, 
				items[i].PubDate,
				items[i].Link,))
			b.Sent[items[i].Link] = true // mark as sent
			count++
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
