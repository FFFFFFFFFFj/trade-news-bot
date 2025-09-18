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

	// обработчики кнопок
	b.Handle(&bot.btnFirst, func(c tb.Context) error {
		chatID := c.Sender().ID
		bot.latestPage[chatID] = 1
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
	b.Handle(&bot.btnNext, func(c tb.Context) error {
		chatID := c.Sender().ID
		bot.latestPage[chatID]++
		bot.ShowLatestNews(chatID, c)
		return nil
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
		row = append(row, b.btnFirst, b.btnPrev)
	}
	if page < totalPages {
		row = append(row, b.btnNext, b.btnLast)
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
// ShowAutopostMenu — меню выбора авторассылки
func (b *Bot) ShowAutopostMenu(chatID int64) {
    times, _ := storage.GetUserAutopost(b.db, chatID)

    msg := "🕒 Настройка авторассылки\n" +
        "Выберите время получения новостей (по МСК).\n" +
        "Максимум 6 раз в день.\n" +
        "Можно также ввести вручную: /autopost 10:30 15:45\n\n" +
        "Сейчас выбрано: "
    if len(times) == 0 {
        msg += "❌ авторассылка отключена"
    } else {
        msg += strings.Join(times, ", ")
    }

    markup := &tb.ReplyMarkup{}
    rows := [][]tb.InlineButton{
        {
            {Text: "❌ Отключить", Data: "autopost:disable"},
        },
    }

    // динамически создаём кнопки по часам 00:00 – 23:00
    var row []tb.InlineButton
    for h := 0; h < 24; h++ {
        t := fmt.Sprintf("%02d:00", h)
        row = append(row, tb.InlineButton{Text: t, Data: "autopost:set:" + t})
        if len(row) == 4 { // по 4 кнопки в ряд
            rows = append(rows, row)
            row = []tb.InlineButton{}
        }
    }
    if len(row) > 0 {
        rows = append(rows, row)
    }

    markup.InlineKeyboard = rows
    b.bot.Send(tb.ChatID(chatID), msg, markup)
}

// HandleAutopost — обработка кнопок выбора времени
func (b *Bot) HandleAutopost(c tb.Context) error {
    data := c.Callback().Data
    userID := c.Sender().ID

    if data == "autopost:disable" {
        _ = storage.SetUserAutopost(b.db, userID, []string{})
        _ = c.Respond(&tb.CallbackResponse{Text: "❌ Автопост отключен"})
        b.ShowAutopostMenu(userID)
        return nil
    }

    if strings.HasPrefix(data, "autopost:set:") {
        t := strings.TrimPrefix(data, "autopost:set:")

        current, _ := storage.GetUserAutopost(b.db, userID)
        // проверка: не больше 6
        if len(current) >= 6 {
            _ = c.Respond(&tb.CallbackResponse{Text: "⚠️ Можно максимум 6"})
            return nil
        }
        // исключаем дубли
        for _, tt := range current {
            if tt == t {
                _ = c.Respond(&tb.CallbackResponse{Text: "⏳ Уже выбрано"})
                return nil
            }
        }

        current = append(current, t)
        _ = storage.SetUserAutopost(b.db, userID, current)
        _ = c.Respond(&tb.CallbackResponse{Text: "✅ Добавлено " + t})
        b.ShowAutopostMenu(userID)
    }
    return nil
}

// StartNewsUpdater запускает авторассылку по расписанию пользователей
func (b *Bot) StartNewsUpdater() {
	loc, _ := time.LoadLocation("Europe/Moscow")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for now := range ticker.C {
		mskNow := now.In(loc)
		hhmm := mskNow.Format("15:04")

		users, _ := storage.GetAllAutopostUsers(b.db)

		for userID, times := range users {
			for _, t := range times {
				if t == hhmm {
					// достаем последние 8 новостей за сегодня
					news, _ := storage.GetTodayNewsPageForUser(b.db, userID, 1, 8)
					if len(news) == 0 {
						continue
					}
					text := "📰 Автоподборка новостей за сегодня:\n\n"
					for _, n := range news {
						text += fmt.Sprintf("• %s\n🔗 %s\n\n", n.Title, n.Link)
					}
					b.SendMessage(userID, text)
				}
			}
		}
	}
}
