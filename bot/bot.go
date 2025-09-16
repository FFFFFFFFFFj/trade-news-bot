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
	bot     *tb.Bot
	db      *sql.DB
	pending map[int64]string
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
		bot:     b,
		db:      db,
		pending: make(map[int64]string),
	}
}

// Start запускает бота и его обработчики
func (b *Bot) Start() {
	// Обработка текстовых сообщений
	b.bot.Handle(tb.OnText, func(c tb.Context) error {
		b.HandleMessage(c.Message())
		return nil
	})

	// Обработка inline-кнопок
	b.bot.Handle(tb.OnCallback, func(c tb.Context) error {
		return b.ToggleSource(c)
	})

	log.Println("🤖 Бот запущен...")
	b.bot.Start()
}

// ShowSourcesMenu отображает пользователю все источники с кнопками подписки/отписки
func (b *Bot) ShowSourcesMenu(chatID int64) error {
	allSources := storage.MustGetAllSources(b.db) // все источники из базы
	userSources, _ := storage.GetUserSources(b.db, chatID) // подписки пользователя

	userSet := make(map[string]bool)
	for _, s := range userSources {
		userSet[s] = true
	}

	var rows [][]tb.InlineButton
	for _, src := range allSources {
		label := src
		if userSet[src] {
			label = "✅ " + label
		} else {
			label = "❌ " + label
		}
		btn := tb.InlineButton{
			Text: label,
			Data: "toggle:" + src,
		}
		rows = append(rows, []tb.InlineButton{btn})
	}

	markup := &tb.ReplyMarkup{InlineKeyboard: rows}
	_, err := b.bot.Send(tb.ChatID(chatID), "Ваши источники:", markup)
	return err
}

// ToggleSource подписывает или отписывает пользователя при нажатии кнопки
func (b *Bot) ToggleSource(c tb.Context) error {
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
			_ = c.Respond(&tb.CallbackResponse{Text: "❌ Отписка"})
		} else {
			_ = storage.Subscribe(b.db, userID, src)
			_ = c.Respond(&tb.CallbackResponse{Text: "✅ Подписка"})
		}

		return b.UpdateSourcesButtons(c)
	}
	return nil
}

// UpdateSourcesButtons обновляет inline-кнопки для источников
func (b *Bot) UpdateSourcesButtons(c tb.Context) error {
	allSources := storage.MustGetAllSources(b.db)
	userSources, _ := storage.GetUserSources(b.db, c.Sender().ID)

	userSet := make(map[string]bool)
	for _, s := range userSources {
		userSet[s] = true
	}

	var rows [][]tb.InlineButton
	for _, src := range allSources {
		label := src
		if userSet[src] {
			label = "✅ " + label
		} else {
			label = "❌ " + label
		}
		btn := tb.InlineButton{
			Text: label,
			Data: "toggle:" + src,
		}
		rows = append(rows, []tb.InlineButton{btn})
	}

	markup := &tb.ReplyMarkup{InlineKeyboard: rows}
	_, err := b.bot.Edit(c.Message(), "Ваши источники:", markup)
	return err
}

// StartNewsUpdater запускает циклическое обновление новостей
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
				_, _ = b.bot.Send(tb.ChatID(userID), msg)
			}
		}
	}
}

// SendMessage отправляет текстовое сообщение пользователю
func (b *Bot) SendMessage(chatID int64, text string) {
	_, err := b.bot.Send(tb.ChatID(chatID), text)
	if err != nil {
		log.Printf("Ошибка отправки: %v", err)
	}
}

// Управление pending действиями (если нужно)
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
