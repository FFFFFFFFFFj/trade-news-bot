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

	return &Bot{
		bot:        b,
		db:         db,
		pending:    make(map[int64]string),
		latestPage: make(map[int64]int),
	}
}

// Start запускает бота
func (b *Bot) Start() {
	// Текстовые команды
	b.bot.Handle(tb.OnText, func(c tb.Context) error {
		b.HandleMessage(c.Message())
		return nil
	})

	// Кнопки подписок
	b.bot.Handle(tb.OnCallback, func(c tb.Context) error {
		data := c.Callback().Data
		if strings.HasPrefix(data, "toggle:") {
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

// SendMessage отправляет текстовое сообщение
func (b *Bot) SendMessage(chatID int64, text string) {
	_, err := b.bot.Send(tb.ChatID(chatID), text)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

// ShowSourcesMenu отображает меню подписок с кнопками
func (b *Bot) ShowSourcesMenu(chatID int64) {
	// создаём пользователя, если нет
	_, _ = b.db.Exec(`INSERT INTO users (id) VALUES ($1) ON CONFLICT DO NOTHING`, chatID)

	allSources := storage.MustGetAllSources(b.db)
	userSources, _ := storage.GetUserSources(b.db, chatID)
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
	_, _ = b.bot.Send(tb.ChatID(chatID), "Ваши источники:", markup)
}

// ToggleSource подписка/отписка при нажатии кнопки
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

		// Обновляем кнопки
		b.ShowSourcesMenu(userID)
	}
	return nil
}

// ShowLatestNews показывает страницу новостей с кнопками навигации
func (b *Bot) ShowLatestNews(chatID int64, c tb.Context) {
	page := b.latestPage[chatID]
	items, _ := storage.GetLatestNewsPageForUser(b.db, chatID, page, 4)

	if len(items) == 0 {
		msg := "⚠️ Больше новостей нет по вашим подпискам."
		if c != nil {
			_, _ = b.bot.Edit(c.Message(), msg)
		} else {
			b.SendMessage(chatID, msg)
		}
		return
	}

	text := "📰 Последние новости по вашим подпискам:\n\n"
	for _, item := range items {
		text += fmt.Sprintf("• %s\n🔗 %s\n\n", item.Title, item.Link)
	}

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
// StartNewsUpdater запускает циклическое обновление новостей
func (b *Bot) StartNewsUpdater(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("🔄 Проверка новостей...")
		newsMap, err := storage.FetchAndStoreNews(b.db)
		if err != nil {
			log.Printf("Ошибка обновления новостей: %v", err)
			continue
		}

		for userID, items := range newsMap {
			for _, item := range items {
				msg := fmt.Sprintf("📰 %s\n🔗 %s\n", item.Title, item.Link)
				_, _ = b.bot.Send(tb.ChatID(userID), msg)
			}
		}
	}
}
