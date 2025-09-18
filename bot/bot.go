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
	latestPage map[int64]int // страница /latest для каждого пользователя

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

	// Обработчики навигации новостей
	b.Handle(&bot.btnFirst, bot.handleFirst)
	b.Handle(&bot.btnPrev, bot.handlePrev)
	b.Handle(&bot.btnNext, bot.handleNext)
	b.Handle(&bot.btnLast, bot.handleLast)

	return bot
}

// Start запускает бота
func (b *Bot) Start() {
	b.bot.Handle(tb.OnText, func(c tb.Context) error {
		b.HandleMessage(c.Message())
		return nil
	})

	b.bot.Handle(tb.OnCallback, func(c tb.Context) error {
		data := c.Callback().Data
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

// Навигация по новостям
func (b *Bot) handleFirst(c tb.Context) error {
	chatID := c.Sender().ID
	b.latestPage[chatID] = 1
	b.ShowLatestNews(chatID, c)
	return nil
}
func (b *Bot) handlePrev(c tb.Context) error {
	chatID := c.Sender().ID
	if b.latestPage[chatID] > 1 {
		b.latestPage[chatID]--
	}
	b.ShowLatestNews(chatID, c)
	return nil
}
func (b *Bot) handleNext(c tb.Context) error {
	chatID := c.Sender().ID
	b.latestPage[chatID]++
	b.ShowLatestNews(chatID, c)
	return nil
}
func (b *Bot) handleLast(c tb.Context) error {
	chatID := c.Sender().ID
	totalCount, _ := storage.GetTodayNewsCountForUser(b.db, chatID)
	pageSize := 4
	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	b.latestPage[chatID] = totalPages
	b.ShowLatestNews(chatID, c)
	return nil
}

// ToggleSource подписка/отписка при нажатии кнопки
func (b *Bot) ToggleSource(c tb.Context) error {
	data := c.Callback().Data
	userID := c.Sender().ID

	src := strings.TrimPrefix(data, "toggle:")
	subs, _ := storage.GetUserSources(b.db, userID)
	isSub := false
	for _, s := range subs {
		if s == src {
			isSub = true
			break
		}
	}

	if isSub {
		_ = storage.Unsubscribe(b.db, userID, src)
		_ = c.Respond(&tb.CallbackResponse{Text: "❌ Отписка"})
	} else {
		_ = storage.Subscribe(b.db, userID, src)
		_ = c.Respond(&tb.CallbackResponse{Text: "✅ Подписка"})
	}

	// Обновляем кнопки без нового сообщения
	b.ShowSourcesMenu(userID, c)
	return nil
}

// HandleAutopost обработка кнопок выбора времени
func (b *Bot) HandleAutopost(c tb.Context) error {
	data := c.Callback().Data
	userID := c.Sender().ID

	if data == "autopost:disable" {
		_ = storage.SetUserAutopost(b.db, userID, []string{})
		_ = c.Respond(&tb.CallbackResponse{Text: "❌ Автопост отключен"})
		b.ShowAutopostMenu(userID, c)
		return nil
	}

	if strings.HasPrefix(data, "autopost:set:") {
		t := strings.TrimPrefix(data, "autopost:set:")
		current, _ := storage.GetUserAutopost(b.db, userID)
		if len(current) >= 6 {
			_ = c.Respond(&tb.CallbackResponse{Text: "⚠️ Можно максимум 6"})
			return nil
		}
		for _, tt := range current {
			if tt == t {
				_ = c.Respond(&tb.CallbackResponse{Text: "⏳ Уже выбрано"})
				return nil
			}
		}
		current = append(current, t)
		_ = storage.SetUserAutopost(b.db, userID, current)
		_ = c.Respond(&tb.CallbackResponse{Text: "✅ Добавлено " + t})
		b.ShowAutopostMenu(userID, c)
	}
	return nil
}
