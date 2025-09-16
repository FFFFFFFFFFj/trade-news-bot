package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/FFFFFFFFFFj/trade-news-bot/bot"
	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
)

func main() {
	// Загружаем .env
	if err := godotenv.Load(); err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN не установлен")
	}

	// Подключаемся к БД
	db, err := storage.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Миграции
	if err := storage.Migrate(db); err != nil {
		log.Fatal("Migration failed:", err)
	}

	// Создаем бота
	b := bot.New(token, db)

	// Проверим наличие источников
	sources, err := storage.GetAllSources(db)
	if err != nil {
		log.Fatalf("Failed to get sources: %v", err)
	}
	if len(sources) == 0 {
		log.Println("No RSS sources found in database. Add sources before starting the bot.")
	}

	// Обновление новостей каждые 10 минут
	b.StartNewsUpdater(10 * time.Minute)

	// Рассылки в 09:00, 15:00, 21:00
	b.StartBroadcastScheduler([]string{"09:00", "15:00", "21:00"}, 8*time.Hour)

	// Ежедневная очистка старых новостей
	b.StartNewsCleaner()

	// Запуск бота (Telebot Start)
	b.Start()
}
