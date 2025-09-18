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

	btnPrev tb.InlineButton
	btnNext tb.InlineButton
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
		btnPrev:    tb.InlineButton{Unique: "latest_prev", Text: "⬅️"},
		btnNext:    tb.InlineButton{Unique: "latest_next", Text: "➡️"},
	}

	// регистрируем обработчики кнопок переключения
	b.Handle(&bot.btnNext, func(c tb.Context) error {
		chatID := c.Sender().ID
		bot.latestPage[chatID]++
		bot.ShowLatestNews(chatID, c)
		return nil
	})
	b.Handle(&bot.btnPrev, func(c tb.Context) error {
		chatID := c.Sender().ID
		if bot.latestPage[chatID] > 1 {
			bot.latestPage[chatID]--
		}
		bot.ShowLatestNews(chatID, c)
		return nil
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

	// Кнопки подписок
	b.bot.Handle(tb.OnCallback, func(c tb.Context) error {
		data := c.Callback().Data
		if strings.HasPrefix(data, "toggle:") {
			return b.ToggleSource(c)
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

// ShowLatestNews показывает страницу новостей по подпискам за сегодня
func (b *Bot) ShowLatestNews(chatID int64, c tb.Context) {
	page := b.latestPage[chatID]
	pageSize := 4

	// всего новостей за сегодня
	totalCount, _ := storage.GetTodayNewsCountForUser(b.db, chatID)

	// считаем количество страниц
	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages == 0 {
		msg := "⚠️ Сегодня новостей по вашим подпискам нет."
		if c != nil {
			_, _ = b.bot.Edit(c.Message(), msg)
		} else {
			b.SendMessage(chatID, msg)
		}
		return
	}

	// если пользователь перелистал дальше
	if page > totalPages {
		b.latestPage[chatID] = totalPages
		page = totalPages
	}

	// получаем новости за сегодня с пагинацией
	items, _ := storage.GetTodayNewsPageForUser(b.db, chatID, page, pageSize)

	if len(items) == 0 {
		msg := "⚠️ Больше новостей за сегодня нет."
		if c != nil {
			_, _ = b.bot.Edit(c.Message(), msg)
		} else {
			b.SendMessage(chatID, msg)
		}
		return
	}

	// формируем текст
	text := fmt.Sprintf("📰 Новости за сегодня (страница %d из %d):\n\n", page, totalPages)
	for _, item := range items {
		text += fmt.Sprintf("• %s\n🔗 %s\n\n", item.Title, item.Link)
	}

	// кнопки
	markup := &tb.ReplyMarkup{}
	var row []tb.InlineButton
	if page > 1 {
		row = append(row, b.btnPrev)
	}
	if page < totalPages {
		row = append(row, b.btnNext)
	}
	if len(row) > 0 {
		markup.InlineKeyboard = [][]tb.InlineButton{row}
	}

	// вывод
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
