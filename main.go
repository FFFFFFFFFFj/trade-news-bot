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
    err := godotenv.Load()
    if err != nil {
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

    // Запускаем обновление новостей каждые 10 минут
    b.StartNewsUpdater(10 * time.Minute)

    // Запускаем рассылки 3 раза в день: 09:00, 15:00, 21:00
    b.StartBroadcastScheduler([]string{"09:00", "15:00", "21:00"}, 8*time.Hour)

    // Ежедневная очистка новостей старше 24 часов
    b.StartNewsCleaner()

    // Запуск обработки входящих сообщений
    b.Start()
}
