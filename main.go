package main

import (
	"log"
	"os"

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
		log.Fatal("Migration faild:", err)
	}

	b := bot.New(token, db)
	b.Start()
}
