package bot

import (
	"log"
	"strings"
	"time"
	"database/sql"

	"github.com/FFFFFFFFFFj/trade-news-bot/rss"
	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
)

type Bot struct {
	Token   string
	APIBase string
	Sent    map[string]bool // cache of sent links
	db      *sql.DB         // connection to the database
}

func New(token string, db *sql.DB) *Bot {
	return &Bot{
		Token:   token,
		APIBase: "https://api.telegram.org/bot" + token + "/",
		Sent:    make(map[string]bool),
		db:      db,
	}
}

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

// Background goroutine for periodic parsing and writing news to the database
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
					err = storage.SaveNews(b.db, item, sourceURL)
					if err != nil {
						log.Printf("SaveNews error: %v", err)
					}
				}
			}
			time.Sleep(interval)
		}
	}()
}
