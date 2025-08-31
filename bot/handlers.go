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
		b.SendMessage(m.Chat.ID, "Hi, I'm bot. Write /latest")
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
			sb.WriteString(fmt.Sprintf("* %s\n%s\n\n", items[i].Title, items[i].Link))
		}

		b.SendMessage(m.Chat.ID, sb.String())

		
	default:
		log.Printf("Got message: %s", txt)
	}
}
