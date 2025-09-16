package bot

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/FFFFFFFFFFj/trade-news-bot/rss"
	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	tb "gopkg.in/telebot.v3"
)

type Bot struct {
	Token          string
	db             *sql.DB
	tbBot          *tb.Bot
	pendingMutex   sync.Mutex
	pendingActions map[int64]string
}

// New создает нового бота
func New(token string, db *sql.DB) *Bot {
	pref := tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	}
	tbBot, err := tb.NewBot(pref)
	if err != nil {
		log.Fatalf("failed to create telebot: %v", err)
	}

	b := &Bot{
		Token:          token,
		db:             db,
		tbBot:          tbBot,
		pendingActions: make(map[int64]string),
	}

	// Глобальный callback для всех inline кнопок
	b.tbBot.Handle(tb.OnCallback, b.HandleInlineCallbacks)

	return b
}

// Pending
func (b *Bot) setPending(userID int64, action string) {
	b.pendingMutex.Lock()
	b.pendingActions[userID] = action
	b.pendingMutex.Unlock()
}

func (b *Bot) getPending(userID int64) (string, bool) {
	b.pendingMutex.Lock()
	act, ok := b.pendingActions[userID]
	b.pendingMutex.Unlock()
	return act, ok
}

func (b *Bot) clearPending(userID int64) {
	b.pendingMutex.Lock()
	delete(b.pendingActions, userID)
	b.pendingMutex.Unlock()
}

// Start запускает Telebot
func (b *Bot) Start() {
	log.Println("Bot started ...")
	b.tbBot.Start()
}

// SendMessage через Telebot
func (b *Bot) SendMessage(chatID int64, text string) error {
	_, err := b.tbBot.Send(&tb.Chat{ID: chatID}, text)
	return err
}

// SendInlineButtons отправляет сообщение с inline-кнопками
func (b *Bot) SendInlineButtons(chatID int64, text string, buttons [][]tb.InlineButton) error {
	markup := &tb.ReplyMarkup{}
	markup.InlineKeyboard = buttons
	_, err := b.tbBot.Send(&tb.Chat{ID: chatID}, text, markup)
	return err
}

// HandleInlineCallbacks обрабатывает нажатия на кнопки
func (b *Bot) HandleInlineCallbacks(c tb.Context) error {
	userID := c.Sender().ID
	sourceURL := c.Callback().Data

	userSources, err := storage.GetUserSources(b.db, userID)
	if err != nil {
		b.SendMessage(userID, "Ошибка при получении ваших подписок.")
		log.Printf("GetUserSources error: %v", err)
		return nil
	}

	if contains(userSources, sourceURL) {
		err = storage.Unsubscribe(b.db, userID, sourceURL)
		if err != nil {
			b.SendMessage(userID, "Не удалось отписаться.")
			log.Printf("Unsubscribe error: %v", err)
			return nil
		}
	} else {
		err = storage.Subscribe(b.db, userID, sourceURL)
		if err != nil {
			b.SendMessage(userID, "Не удалось подписаться.")
			log.Printf("Subscribe error: %v", err)
			return nil
		}
	}

	// Перестраиваем inline-кнопки
	allSources, _ := storage.GetAllSources(b.db)
	userSources, _ = storage.GetUserSources(b.db, userID)

	var buttons [][]tb.InlineButton
	for _, src := range allSources {
		displayName := src
		if u, err := url.Parse(src); err == nil {
			displayName = u.Host
		}
		prefix := ""
		if contains(userSources, src) {
			prefix = "✅ "
		}
		btn := tb.InlineButton{
			Unique: "toggle_" + displayName,
			Text:   prefix + displayName,
			Data:   src,
		}
		buttons = append(buttons, []tb.InlineButton{btn})
	}

	markup := &tb.ReplyMarkup{InlineKeyboard: buttons}
	c.Edit("Ваши подписки:", markup)

	// Убираем "часики"
	c.Respond()
	return nil
}

// StartNewsUpdater обновляет новости каждые interval
func (b *Bot) StartNewsUpdater(interval time.Duration) {
	go func() {
		for {
			sources, err := storage.GetAllSources(b.db)
			if err != nil {
				log.Printf("Failed to get sources: %v", err)
				time.Sleep(time.Minute)
				continue
			}
			for _, src := range sources {
				items, err := rss.Fetch(src)
				if err != nil {
					log.Printf("RSS fetch error (%s): %v", src, err)
					continue
				}
				for _, it := range items {
					if err := storage.SaveNews(b.db, it, src); err != nil {
						log.Printf("SaveNews error: %v", err)
					}
				}
			}
			time.Sleep(interval)
		}
	}()
}

// StartBroadcastScheduler
func (b *Bot) StartBroadcastScheduler(schedule []string, since time.Duration) {
	go func() {
		for {
			now := time.Now().Format("15:04")
			for _, t := range schedule {
				if now == t {
					sinceTime := time.Now().Add(-since)
					users, err := storage.GetUsersWithSubscriptions(b.db)
					if err != nil {
						log.Printf("GetUsersWithSubscriptions error: %v", err)
						continue
					}
					for _, uid := range users {
						items, err := storage.GetRecentNewsForUser(b.db, uid, sinceTime)
						if err != nil {
							log.Printf("GetRecentNewsForUser error for %d: %v", uid, err)
							continue
						}
						for _, it := range items {
							msg := fmt.Sprintf("📰 %s\n🕒 %s\n🔗 %s\n\n", it.Title, it.PubDate, it.Link)
							_ = b.SendMessage(uid, msg)
						}
					}
				}
			}
			time.Sleep(time.Minute)
		}
	}()
}

// StartNewsCleaner
func (b *Bot) StartNewsCleaner() {
	go func() {
		for {
			_, err := b.db.Exec("DELETE FROM news WHERE pub_date < NOW() - INTERVAL '24 hours'")
			if err != nil {
				log.Printf("Clean old news error: %v", err)
			}
			time.Sleep(24 * time.Hour)
		}
	}()
}

// вспомогательная функция
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
