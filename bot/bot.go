package bot

import (
	"database/sql"
	"fmt"
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

// StartBroadcastScheduler запускает рассылки в заданные часы (в локальном времени сервера).
// schedule - список строк в формате "HH:MM" (например: []string{"09:00","15:00","21:00"})
// since - интервал (например 8*time.Hour) — будем брать новости, старше которых нет смысла рассылать.
func (b *Bot) StartBroadcastScheduler(schedule []string, since time.Duration) {
	go func() {
		for {
			now := time.Now().Format("15:04")
			for _, t := range schedule {
				if now == t {
					// вычислим границу sinceTime
					sinceTime := time.Now().Add(-since)
					// получим список пользователей с подписками
					users, err := storage.GetUsersWithSubscriptions(b.db)
					if err != nil {
						log.Printf("GetUsersWithSubscriptions error: %v", err)
						continue
					}
					for _, uid := range users {
						items, err := storage.GetRecentNewsForUser(b.db, uid, sinceTime)
						if err != nil {
							log.Printf("GetRecentNewsForUser error for %d: %v", uid, err)
							continue
						}
						if len(items) == 0 {
							continue
						}
						for _, it := range items {
							msg := fmt.Sprintf("📰 %s\n🕒 %s\n🔗 %s\n\n", it.Title, it.PubDate, it.Link)
							_ = b.SendMessage(uid, msg) // игнорируем ошибку отправки для одной конкретной — лог в SendMessage
						}
					}
				}
			}
			time.Sleep(60 * time.Second) // проверяем каждую минуту
		}
	}()
}

// StartNewsCleaner удаляет все новости старше 24 часов (один раз в день)
func (b *Bot) StartNewsCleaner() {
	go func() {
		for {
			_, err := b.db.Exec("DELETE FROM news WHERE pub_date < NOW() - INTERVAL '24 hours'")
			if err != nil {
				log.Printf("Clean old news error: %v", err)
			}
			// запускать раз в 24 часа
			time.Sleep(24 * time.Hour)
		}
	}()
}
