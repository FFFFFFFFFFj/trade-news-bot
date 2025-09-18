package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	tb "gopkg.in/telebot.v3"
)

type Bot struct {
	bot        *tb.Bot
	db         *sql.DB
	pending    map[int64]string
	latestPage map[int64]int

	btnFirst tb.InlineButton
	btnPrev  tb.InlineButton
	btnNext  tb.InlineButton
	btnLast  tb.InlineButton
}

// New создаёт нового бота
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

	// Навигация по /latest
	b.Handle(&botInstance.btnFirst, func(c tb.Context) error {
		chatID := c.Sender().ID
		botInstance.latestPage[chatID] = 1
		return botInstance.ShowLatestNews(chatID, c)
	})
	b.Handle(&botInstance.btnPrev, func(c tb.Context) error {
		chatID := c.Sender().ID
		if botInstance.latestPage[chatID] > 1 {
			botInstance.latestPage[chatID]--
		}
		return botInstance.ShowLatestNews(chatID, c)
	})
	b.Handle(&botInstance.btnNext, func(c tb.Context) error {
		chatID := c.Sender().ID
		botInstance.latestPage[chatID]++
		return botInstance.ShowLatestNews(chatID, c)
	})
	b.Handle(&botInstance.btnLast, func(c tb.Context) error {
		chatID := c.Sender().ID
		totalCount, _ := storage.GetTodayNewsCountForUser(botInstance.db, chatID)
		pageSize := 4
		totalPages := (totalCount + pageSize - 1) / pageSize
		if totalPages < 1 {
			totalPages = 1
		}
		botInstance.latestPage[chatID] = totalPages
		return botInstance.ShowLatestNews(chatID, c)
	})

	return botInstance
}

// Start запускает бота
func (b *Bot) Start() {
	b.bot.Handle(tb.OnText, func(c tb.Context) error {
		b.HandleMessage(c.Message())
		return nil
	})

	// Обработчики кнопок
	b.bot.Handle(tb.OnCallback, func(c tb.Context) error {
		data := c.Callback().Data
		userID := c.Sender().ID
		if strings.HasPrefix(data, "toggle:") {
			return b.ToggleSource(c)
		}
		if strings.HasPrefix(data, "autopost:") {
			return b.HandleAutopost(c)
		}
		return nil
	})

	log.Println("🤖 Бот запущен...")
	b.bot.Start()
}

// SendMessage отправляет текстовое сообщение
func (b *Bot) SendMessage(chatID int64, text string) {
	_, err := b.bot.Send(tb.ChatID(chatID), text)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}
