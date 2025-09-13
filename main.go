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

	// News sources
	sources := []string{
		"https://www.finmarket.ru/rss/main.asp",
		"https://www.finmarket.ru/rss/ecworld.asp",
		"https://www.finmarket.ru/rss/finances.asp",
		"https://www.bfm.ru/rss/news.xml",
		"https://rssexport.rbc.ru/rbcnews/economics/full.rss",
		"https://rssexport.rbc.ru/rbcnews/finance/full.rss",
		"https://rssexport.rbc.ru/rbcnews/business/full.rss",
		"https://www.interfax.ru/rss.asp",
		"https://tass.ru/rss/v2/economy.xml",
		"https://ru.investing.com/rss/news.rss",
		"https://ru.investing.com/rss/forex.rss",
		"https://ru.investing.com/rss/cryptocurrency.rss",
	}

	// Run background news update (every 10 minutes)
	b.StartNewsUpdater(sources, 10*time.Minute)

	b.Start()
}
