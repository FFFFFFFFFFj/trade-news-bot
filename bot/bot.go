package bot

import (
	"database/sql"
	"log"
	"time"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	tb "gopkg.in/telebot.v3"
)

type Bot struct {
	bot        *tb.Bot
	db         *sql.DB
	pending    map[int64]string
	latestPage map[int64]int

	// кнопки навигации /latest
	btnFirst tb.InlineButton
	btnPrev  tb.InlineButton
	btnNext  tb.InlineButton
	btnLast  tb.InlineButton
}

func New(token string, db *sql.DB) *Bot {
	pref := tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tb.NewBot(pref)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	botInstance := &Bot{
		bot:        b,
		db:         db,
		pending:    make(map[int64]string),
		latestPage: make(map[int64]int),

		btnFirst: tb.InlineButton{Unique: "latest_first", Text: "⏮"},
		btnPrev:  tb.InlineButton{Unique: "latest_prev", Text: "⬅️"},
		btnNext:  tb.InlineButton{Unique: "latest_next", Text: "➡️"},
		btnLast:  tb.InlineButton{Unique: "latest_last", Text: "⏭"},
	}

	// Навигация /latest
	botInstance.bot.Handle(&botInstance.btnFirst, func(c tb.Context) error {
		chatID := c.Sender().ID
		botInstance.latestPage[chatID] = 1
		botInstance.ShowLatestNews(chatID, c)
		return nil
	})
	botInstance.bot.Handle(&botInstance.btnPrev, func(c tb.Context) error {
		chatID := c.Sender().ID
		if botInstance.latestPage[chatID] > 1 {
			botInstance.latestPage[chatID]--
		}
		botInstance.ShowLatestNews(chatID, c)
		return nil
	})
	botInstance.bot.Handle(&botInstance.btnNext, func(c tb.Context) error {
		chatID := c.Sender().ID
		botInstance.latestPage[chatID]++
		botInstance.ShowLatestNews(chatID, c)
		return nil
	})
	botInstance.bot.Handle(&botInstance.btnLast, func(c tb.Context) error {
		chatID := c.Sender().ID
		totalCount, _ := storage.GetTodayNewsCountForUser(botInstance.db, chatID)
		pageSize := 4
		totalPages := (totalCount + pageSize - 1) / pageSize
		if totalPages < 1 {
			totalPages = 1
		}
		botInstance.latestPage[chatID] = totalPages
		botInstance.ShowLatestNews(chatID, c)
		return nil
	})

	// Текстовые сообщения
	botInstance.bot.Handle(tb.OnText, func(c tb.Context) error {
		botInstance.HandleMessage(c.Message())
		return nil
	})

	return botInstance
}

func (b *Bot) Start() {
	b.bot.Start()
}

func (b *Bot) SendMessage(chatID int64, text string) {
	_, _ = b.bot.Send(tb.ChatID(chatID), text)
}

// Проверка на админа
var AdminIDs = map[int64]bool{
	839986298: true, // твой ID
}

func (b *Bot) IsAdmin(userID int64) bool {
	return AdminIDs[userID]
}

func (b *Bot) StartNewsUpdater() {
    ticker := time.NewTicker(10 * time.Minute) // интервал обновления
    for range ticker.C {
        newsMap, err := storage.FetchAndStoreNews(b.db)
        if err != nil {
            log.Printf("Ошибка обновления новостей: %v", err)
            continue
        }

        for userID, newsItems := range newsMap {
            for _, n := range newsItems {
                b.SendMessage(userID, fmt.Sprintf("📰 %s\n%s", n.Title, n.Link))
            }
        }
    }
}
