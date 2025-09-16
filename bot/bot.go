package bot

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	tb "gopkg.in/telebot.v3"
)

type Bot struct {
	bot     *tb.Bot
	db      *sql.DB
	pending map[int64]string
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

	return &Bot{
		bot:     b,
		db:      db,
		pending: make(map[int64]string),
	}
}

func (b *Bot) Start() {
	// Обработка текстовых сообщений
	b.bot.Handle(tb.OnText, func(c tb.Context) error {
		return b.HandleMessage(c.Message())
	})

	// Обработка inline-кнопок
	b.bot.Handle(&tb.Callback{Data: tb.Any}, func(c tb.Context) error {
		data := c.Callback().Data
		userID := c.Sender().ID

		if strings.HasPrefix(data, "toggle:") {
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
				c.Respond(&tb.CallbackResponse{Text: "❌ Отписка"})
			} else {
				_ = storage.Subscribe(b.db, userID, src)
				c.Respond(&tb.CallbackResponse{Text: "✅ Подписка"})
			}

			return b.UpdateSourcesButtons(c)
		}

		return nil
	})

	log.Println("🤖 Бот запущен...")
	b.bot.Start()
}

// Обновление inline-кнопок источников
func (b *Bot) UpdateSourcesButtons(c tb.Context) error {
	allSources, _ := storage.GetAllSources(b.db)
	userSources, _ := storage.GetUserSources(b.db, c.Sender().ID)

	userSet := make(map[string]bool)
	for _, s := range userSources {
		userSet[s] = true
	}

	var rows [][]tb.InlineButton
	for _, src := range allSources {
		u, _ := url.Parse(src)
		label := u.Host
		if userSet[src] {
			label = "✅ " + label
		}
		btn := tb.InlineButton{
			Text: label,
			Data: "toggle:" + src,
		}
		rows = append(rows, []tb.InlineButton{btn})
	}

	markup := &tb.ReplyMarkup{InlineKeyboard: rows}
	return b.bot.Edit(c.Message(), "Ваши источники:", markup)
}

// Обновление новостей в фоне
func (b *Bot) StartNewsUpdater(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("🔄 Проверка новостей...")
		news, err := storage.FetchAndStoreNews(b.db)
		if err != nil {
			log.Printf("Ошибка обновления новостей: %v", err)
			continue
		}

		for userID, items := range news {
			for _, item := range items {
				msg := fmt.Sprintf("📰 %s\n🔗 %s\n", item.Title, item.Link)
				b.SendMessage(userID, msg)
			}
		}
	}
}

// Отправка сообщения пользователю
func (b *Bot) SendMessage(chatID int64, text string) {
	err := b.bot.Send(tb.ChatID(chatID), text)
	if err != nil {
		log.Printf("Ошибка отправки: %v", err)
	}
}

// Методы для pending действий (например, подписки)
func (b *Bot) setPending(chatID int64, action string) {
	b.pending[chatID] = action
}

func (b *Bot) getPending(chatID int64) (string, bool) {
	action, ok := b.pending[chatID]
	return action, ok
}

func (b *Bot) clearPending(chatID int64) {
	delete(b.pending, chatID)
}
