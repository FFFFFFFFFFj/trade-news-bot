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

	// кнопки для навигации
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

	bot := &Bot{
		bot:        b,
		db:         db,
		pending:    make(map[int64]string),
		latestPage: make(map[int64]int),

		btnFirst: tb.InlineButton{Unique: "latest_first", Text: "⏮"},
		btnPrev:  tb.InlineButton{Unique: "latest_prev", Text: "⬅️"},
		btnNext:  tb.InlineButton{Unique: "latest_next", Text: "➡️"},
		btnLast:  tb.InlineButton{Unique: "latest_last", Text: "⏭"},
	}

	// кнопки пагинации новостей
	b.Handle(&bot.btnFirst, func(c tb.Context) error {
		chatID := c.Sender().ID
		bot.latestPage[chatID] = 1
		return bot.ShowLatestNews(chatID, c)
	})
	b.Handle(&bot.btnPrev, func(c tb.Context) error {
		chatID := c.Sender().ID
		if bot.latestPage[chatID] > 1 {
			bot.latestPage[chatID]--
		}
		return bot.ShowLatestNews(chatID, c)
	})
	b.Handle(&bot.btnNext, func(c tb.Context) error {
		chatID := c.Sender().ID
		bot.latestPage[chatID]++
		return bot.ShowLatestNews(chatID, c)
	})
	b.Handle(&bot.btnLast, func(c tb.Context) error {
		chatID := c.Sender().ID
		totalCount, _ := storage.GetTodayNewsCountForUser(bot.db, chatID)
		pageSize := 4
		totalPages := (totalCount + pageSize - 1) / pageSize
		if totalPages < 1 {
			totalPages = 1
		}
		bot.latestPage[chatID] = totalPages
		return bot.ShowLatestNews(chatID, c)
	})

	return bot
}

// Start запускает бота
func (b *Bot) Start() {
	// Текстовые команды
	b.bot.Handle(tb.OnText, func(c tb.Context) error {
		b.HandleMessage(c.Message())
		return nil
	})

	// Кнопки подписок и автопост
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
