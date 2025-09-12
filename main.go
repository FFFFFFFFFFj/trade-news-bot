package main

import (
	"log"
	"os"

	"github.com/FFFFFFFFFFj/trade-news-bot/bot"
)


func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN not set")
	}

	db, err := ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = Migrate(db)
	if err != nil {
		log.Fatal("Migration faild:", err)
	}

	b := bot.New(token, db)
	b.Start()
}
