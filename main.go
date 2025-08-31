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

	b := bot.New(token)
	b.Start()
}
