package bot

import (
	"fmt"
	"log"
	"strings"
	"time"
	"database/sql"

	"github.com/FFFFFFFFFFj/trade-news-bot/rss"
	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
)

type Bot struct {
	Token   string
	APIBase string
	Sent    map[string]bool // кэш отправленных ссылок (для одного запуска)
	db      *sql.DB         // подключение к базе
}

var AdminIDs = map[int64]bool{
	839986298: true,
}

func (b *Bot) IsAdmin(userID int64) bool {
	return AdminIDs[userID]
}

func New(token string, db *sql.DB) *Bot {
	return &Bot{
		Token:   token,
		APIBase: "https://api.telegram.org/bot" + token + "/",
		Sent:    make(map[string]bool),
		db:      db,
	}
}

func (b *Bot) Start() {
	log.Println("Bot started ...")
	var offset int
	for {
		updates, err := b.GetUpdates(offset, 30)
		if err != nil {
			if strings.Contains(err.Error(), "Client.Timeout") {
				continue
			}
			log.Printf("getUpdates error: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			if u.Message != nil {
				b.HandleMessage(u.Message)
			}
		}
	}
}

func (b *Bot) StartNewsUpdater(sources []string, interval time.Duration) {
	go func() {
		for {
			for _, sourceURL := range sources {
				items, err := rss.Fetch(sourceURL)
				if err != nil {
					log.Printf("RSS fetch error (%s): %v", sourceURL, err)
					continue
				}
				for _, item := range items {
					err = storage.SaveNews(b.db, item, sourceURL)
					if err != nil {
						log.Printf("SaveNews error: %v", err)
					}
				}
			}
			time.Sleep(interval)
		}
	}()
}

// Обработка сообщений
func (b *Bot) HandleMessage(m *Message) {
	txt := strings.TrimSpace(m.Text)
	switch {
	case txt == "/start":
		b.SendMessage(m.Chat.ID, "👋 Приветствую! Я — ваш бот для получения свежих новостей с инвестиционных сайтов 📈📰.\n\n"+
			"⚡ Чтобы узнать все возможности и как пользоваться ботом, отправьте команду\n"+
			"👉 /help\n\n"+
			"Держите руку на пульсе финансового мира вместе со мной! 🚀💰")
	case txt == "/help":
		helpText := "/start - запустить бота\n" +
			"/latest - последние новости\n" +
			"/help - список команд\n" +
			"/addsource <URL> - добавить источник (админ)\n" +
			"/removesource <URL> - удалить источник (админ)\n" +
			"/listsources - показать все источники (админ)"
		b.SendMessage(m.Chat.ID, helpText)
	case txt == "/latest":
		limit := 5
		items, err := storage.GetUnreadNews(b.db, m.Chat.ID, limit)
		if err != nil {
			b.SendMessage(m.Chat.ID, "⚠️ Не удалось загрузить новости из базы данных.")
			log.Printf("GetUnreadNews error: %v", err)
			return
		}
		if len(items) == 0 {
			b.SendMessage(m.Chat.ID, "🚫 Сейчас в базе нет свежих новостей для вас.")
			return
		}

		var sb strings.Builder
		count := 0
		for _, item := range items {
			if b.Sent[item.Link] {
				continue // уже отправили за время работы бота
			}

			sb.Reset()
			sb.WriteString(fmt.Sprintf("📌 %s\n🕒 %s\n🔗 %s\n\n",
				item.Title,
				item.PubDate,
				item.Link))

			err = b.SendMessage(m.Chat.ID, sb.String())
			if err != nil {
				log.Printf("SendMessage error: %v", err)
				continue
			}

			// Отмечаем новость как прочитанную для этого пользователя
			if err := storage.MarkNewsAsRead(b.db, m.Chat.ID, item.Link); err != nil {
				log.Printf("MarkNewsAsRead error: %v", err)
			}

			b.Sent[item.Link] = true
			count++
		}
		if count == 0 {
			b.SendMessage(m.Chat.ID, "🚫 Пока нет новых новостей для отправки.")
			return
		}
    // Можно отправить итоговое сообщение или оставить пустым
	case strings.HasPrefix(txt, "/addsource"):
		// обработка команды добавления источника (админ)
	case strings.HasPrefix(txt, "/removesource"):
		// обработка команды удаления источника (админ)
	case txt == "/listsources":
		// показ всех источников (админ)
	default:
		log.Printf("Got message: %s", txt)
	}
}
