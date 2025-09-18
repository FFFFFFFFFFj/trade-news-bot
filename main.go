package main
//
import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/FFFFFFFFFFj/trade-news-bot/bot"
	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN не установлен")
	}

	db, err := storage.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := storage.Migrate(db); err != nil {
		log.Fatal("Migration failed:", err)
	}

	b := bot.New(token, db)

	go b.StartNewsUpdater()
	b.Start()
}
