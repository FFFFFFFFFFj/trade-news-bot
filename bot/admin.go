package bot

import (
	"log"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
)

func (b *Bot) AdminBroadcast(msg string) {
	users, err := storage.GetAllUsers(b.db)
	if err != nil {
		log.Printf("Ошибка выборки пользователей: %v", err)
		return
	}

	for _, u := range users {
		b.SendMessage(u, "📢 "+msg)
	}
}
