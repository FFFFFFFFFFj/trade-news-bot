package main

import (
	"log"
	"os"
	"time"
	"github.com/FFFFFFFFFFj/trade-news-bot/bot"
	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN not set")
	}

	db, err := storage.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = storage.Migrate(db)
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	b := bot.New(token, db)

	sources, err := storage.GetAllSources(db)
	if err != nil {
		log.Fatalf("Failed to get sources: %v", err)
	}
	if len(sources) == 0 {
		log.Fatal("No RSS sources found in database. Add sources before starting the bot.")
	}

	// Запускаем обновление новостей (внутри уже запускается goroutine)
	b.StartNewsUpdater(sources, 10*time.Minute)

	// Запускаем рассылки 3 раза в день: пример 09:00, 15:00, 21:00 — и берём новости за последние 8 часов
	b.StartBroadcastScheduler([]string{"09:00", "15:00", "21:00"}, 8*time.Hour)

	// Ежедневная очистка старых новостей
	b.StartNewsCleaner()

	// Блокирующий цикл обработки входящих сообщений
	b.Start()
}
