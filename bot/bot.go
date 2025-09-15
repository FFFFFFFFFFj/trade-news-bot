ppackage bot

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	"github.com/FFFFFFFFFFj/trade-news-bot/rss"
)

// Bot представляет структуру Telegram-бота
type Bot struct {
	Token   string
	APIBase string
	db      *sql.DB
}

// New создает и возвращает новый экземпляр бота
func New(token string, db *sql.DB) *Bot {
	return &Bot{
		Token:   token,
		APIBase: "https://api.telegram.org/bot" + token + "/",
		db:      db,
	}
}

// Start запускает цикл получения обновлений от Telegram
func (b *Bot) Start() {
	log.Println("Bot started ...")
	var offset int
	for {
		updates, err := b.GetUpdates(offset, 30)
		if err != nil {
			if strings.Contains(err.Error(), "Client.Timeout") {
				continue
			}
			log.Printf("getUpdates error: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			if u.Message != nil {
				b.HandleMessage(u.Message)
			}
		}
	}
}

// StartNewsUpdater запускает периодическое обновление новостей
func (b *Bot) StartNewsUpdater(sources []string, interval time.Duration) {
	go func() {
		for {
			for _, sourceURL := range sources {
				items, err := rss.Fetch(sourceURL)
				if err != nil {
					log.Printf("RSS fetch error (%s): %v", sourceURL, err)
					continue
				}
				for _, item := range items {
					if err := storage.SaveNews(b.db, item, sourceURL); err != nil {
						log.Printf("SaveNews error: %v", err)
					}
				}
			}
			time.Sleep(interval)
		}
	}()
}
