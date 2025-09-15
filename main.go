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

	// We get sources from the database
	sources, err := storage.GetAllSources(db)
	if err != nil {
		log.Fatalf("Failed to get sources: %v", err)
	}
	if len(sources) == 0 {
		log.Fatal("No RSS sources found in database. Add sources before starting the bot.")
	}

	b.StartNewsUpdater(sources, 10*time.Minute)

	b.Start()
}
