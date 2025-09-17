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
}

// New создает нового бота
func New(token string, db *sql.DB) *Bot {
	pref := tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tb.NewBot(pref)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	return &Bot{
		bot:        b,
		db:         db,
		pending:    make(map[int64]string),
		latestPage: make(map[int64]int),
	}
}

// Start запускает бота и его обработчики
func (b *Bot) Start() {
	// Текстовые команды
	b.bot.Handle(tb.OnText, func(c tb.Context) error {
		b.HandleMessage(c.Message())
		return nil
	})

	// Кнопки подписок
	b.bot.Handle(tb.OnCallback, func(c tb.Context) error {
		if strings.HasPrefix(c.Callback().Data, "toggle:") {
			return b.ToggleSource(c)
		}
		return nil
	})

	// Кнопки навигации новостей
	b.bot.Handle(&tb.InlineButton{Data: "latest_next"}, func(c tb.Context) error {
		chatID := c.Sender().ID
		b.latestPage[chatID]++
		b.ShowLatestNews(chatID, c)
		return nil
	})

	b.bot.Handle(&tb.InlineButton{Data: "latest_prev"}, func(c tb.Context) error {
		chatID := c.Sender().ID
		if b.latestPage[chatID] > 1 {
			b.latestPage[chatID]--
		}
		b.ShowLatestNews(chatID, c)
		return nil
	})

	log.Println("🤖 Бот запущен...")
	b.bot.Start()
}

// Показывает страницу новостей (по активным подпискам)
func (b *Bot) ShowLatestNews(chatID int64, c tb.Context) {
	page := b.latestPage[chatID]

	// Берем только активные подписки
	items, _ := storage.GetLatestNewsPageForUser(b.db, chatID, page, 4)

	if len(items) == 0 {
		if c != nil {
			_, _ = b.bot.Edit(c.Message(), "⚠️ Больше новостей нет по вашим подпискам.")
		} else {
			b.SendMessage(chatID, "⚠️ Больше новостей нет по вашим подпискам.")
		}
		return
	}

	text := "📰 Последние новости по вашим подпискам:\n\n"
	for _, item := range items {
		text += fmt.Sprintf("• %s\n🔗 %s\n\n", item.Title, item.Link)
	}

	// Кнопки
	prevBtn := tb.InlineButton{Text: "⬅️", Data: "latest_prev"}
	nextBtn := tb.InlineButton{Text: "➡️", Data: "latest_next"}
	markup := &tb.ReplyMarkup{}

	if page > 1 {
		markup.InlineKeyboard = [][]tb.InlineButton{{prevBtn, nextBtn}}
	} else {
		markup.InlineKeyboard = [][]tb.InlineButton{{nextBtn}}
	}

	if c != nil {
		_, _ = b.bot.Edit(c.Message(), text, markup)
	} else {
		_, _ = b.bot.Send(tb.ChatID(chatID), text, markup)
	}
}
