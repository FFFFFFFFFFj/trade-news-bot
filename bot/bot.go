package bot

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	tb "gopkg.in/telebot.v3"
)

type Bot struct {
	tb         *tb.Bot
	db         *sql.DB
	latestPage map[int64]int
	pendingDel map[int64]string // ожидаемое удаление источника
	waitingAdd map[int64]bool   // ожидаем ввод нового источника
}

func NewBot(telegramToken string, db *sql.DB) (*Bot, error) {
	b, err := tb.NewBot(tb.Settings{
		Token: telegramToken,
	})
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		tb:         b,
		db:         db,
		latestPage: make(map[int64]int),
		pendingDel: make(map[int64]string),
		waitingAdd: make(map[int64]bool),
	}

	b.Handle(tb.OnText, func(m *tb.Message) {
		bot.HandleMessage(m)
	})

	b.Handle(&tb.Callback{}, func(c *tb.Callback) {
		bot.HandleCallback(c)
	})

	return bot, nil
}

func (b *Bot) Start() {
	b.tb.Start()
}

func (b *Bot) SendMessage(chatID int64, text string, opts ...interface{}) {
	_, err := b.tb.Send(&tb.Chat{ID: chatID}, text, opts...)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

// ==== Админ-меню ====
func (b *Bot) ShowAdminMenu(chatID int64) {
	menu := &tb.ReplyMarkup{}
	btnSources := menu.Data("📂 Управлять источниками", "admin_sources")
	btnPost := menu.Data("📢 Сделать рассылку", "admin_post")

	menu.Inline(
		menu.Row(btnSources),
		menu.Row(btnPost),
	)

	b.SendMessage(chatID, "⚙️ Админ-меню:", menu)
}

func (b *Bot) ShowSourcesAdmin(chatID int64) {
	sources := storage.MustGetAllSources(b.db)

	menu := &tb.ReplyMarkup{}
	var rows []tb.Row
	for _, src := range sources {
		btnDel := menu.Data("❌ "+src.URL, "del_source", src.URL)
		rows = append(rows, menu.Row(btnDel))
	}
	btnAdd := menu.Data("➕ Добавить источник", "add_source")
	rows = append(rows, menu.Row(btnAdd))

	menu.Inline(rows...)
	b.SendMessage(chatID, "📂 Источники:", menu)
}

func (b *Bot) BroadcastMessage(content string) {
	users, err := storage.GetAllUsers(b.db)
	if err != nil {
		log.Printf("Ошибка получения пользователей: %v", err)
		return
	}
	for _, uid := range users {
		b.SendMessage(uid, "📢 "+content)
	}
}

// ==== Callbacks ====
func (b *Bot) HandleCallback(c *tb.Callback) {
	data := c.Data
	chatID := c.Sender.ID

	switch {
	case data == "admin_sources":
		b.ShowSourcesAdmin(chatID)

	case data == "add_source":
		b.waitingAdd[chatID] = true
		b.SendMessage(chatID, "✍️ Введите URL нового источника:")

	case data == "admin_post":
		b.SendMessage(chatID, "Используйте команду:\n/post текст_сообщения")

	case c.Unique == "del_source":
		url := c.Data
		b.pendingDel[chatID] = url

		menu := &tb.ReplyMarkup{}
		btnYes := menu.Data("✅ Да", "confirm_del_yes", url)
		btnNo := menu.Data("❌ Нет", "confirm_del_no", url)
		menu.Inline(menu.Row(btnYes, btnNo))

		b.SendMessage(chatID, fmt.Sprintf("Удалить источник?\n%s", url), menu)

	case c.Unique == "confirm_del_yes":
		url := c.Data
		err := storage.DeleteSource(b.db, url)
		if err != nil {
			b.SendMessage(chatID, "❌ Ошибка удаления: "+err.Error())
		} else {
			b.SendMessage(chatID, "🗑 Источник удалён: "+url)
		}
		delete(b.pendingDel, chatID)
		b.ShowSourcesAdmin(chatID)

	case c.Unique == "confirm_del_no":
		delete(b.pendingDel, chatID)
		b.SendMessage(chatID, "❎ Удаление отменено")
		b.ShowSourcesAdmin(chatID)
	}
}
