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
		b.SendMessage(m.Chat.ID, "ПриветX ")
	
	case "/help":
		helpText := "/start - ***\n" +
					"/latest - ***\n" +
					"/help - ***"
		b.SendMessage(m.Chat.ID, helpText) 
		
	case "/latest":
		items, err := rss.Fetch("https://www.investing.com/rss/news.rss")
		if err != nil {
			b.SendMessage(m.Chat.ID, "Error loading news : (")
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
			sb.WriteString(fmt.Sprintf("* %s\n*%s\n*%s\n\n", 
				items[i].Title, 
				items[i].PubDate,
				items[i].Link,))
			b.Sent[items[i].Link] = true // mark as sent
		}

		if sb.Len() == 0 {
			b.SendMessage(m.Chat.ID, "No new news at the moment.")
			return
		}

		b.SendMessage(m.Chat.ID, sb.String())

		
	default:
		log.Printf("Got message: %s", txt)
	}
}
