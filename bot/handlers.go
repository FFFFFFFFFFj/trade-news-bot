package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/FFFFFFFFFFj/trade-news-bot/storage"
	tb "gopkg.in/telebot.v3"
)

func (b *Bot) HandleMessage(m *tb.Message) {
	_, _ = b.db.Exec(`INSERT INTO users (id) VALUES ($1) ON CONFLICT DO NOTHING`, m.Chat.ID)
	txt := strings.TrimSpace(m.Text)
	userID := m.Chat.ID

	// Проверка режима ввода админских команд
	if mode, ok := b.pending[userID]; ok && b.IsAdmin(userID) {
		switch mode {
		case "addsource":
			if txt == "" {
				b.SendMessage(userID, "⚠️ URL пустой")
			} else if err := storage.AddSource(b.db, txt); err != nil {
				b.SendMessage(userID, "❌ Ошибка добавления источника")
			} else {
				b.SendMessage(userID, "✅ Источник добавлен: "+txt)
			}
			b.pending[userID] = ""
			return

		case "removesource":
			if txt == "" {
				b.SendMessage(userID, "⚠️ URL пустой")
			} else if err := storage.RemoveSource(b.db, txt); err != nil {
				b.SendMessage(userID, "❌ Ошибка удаления источника")
			} else {
				b.SendMessage(userID, "✅ Источник удалён: "+txt)
			}
			b.pending[userID] = ""
			return

		case "broadcast":
			if txt == "" {
				b.SendMessage(userID, "⚠️ Сообщение пустое")
			} else {
				b.AdminBroadcast(txt)
			}
			b.pending[userID] = ""
			return

		case "setchannel":
			if txt == "" {
				b.SendMessage(userID, "⚠️ Ссылка пустая")
			} else {
				_ = storage.SetSetting(b.db, "channel", txt)
				b.SendMessage(userID, "✅ Ссылка на канал обновлена")
			}
			b.pending[userID] = ""
			return

		case "setmanual":
			if txt == "" {
				b.SendMessage(userID, "⚠️ Ссылка пустая")
			} else {
				_ = storage.SetSetting(b.db, "manual", txt)
				b.SendMessage(userID, "✅ Ссылка на инструкцию обновлена")
			}
			b.pending[userID] = ""
			return
		}
	}

	switch {
	case txt == "/start":
		if b.IsAdmin(userID) {
			usersCount, _ := storage.GetUsersCount(b.db)
			activeUsers, _ := storage.GetActiveUsersCount(b.db)
			autopostUsers, _ := storage.GetAutopostUsersCount(b.db)
			msg := fmt.Sprintf("👑 Админ\nID: %d\nВсего пользователей: %d\nПодписанных: %d\nС автопостом: %d\nВсего источников: %d",
				userID, usersCount, activeUsers, autopostUsers, len(storage.MustGetAllSources(b.db)))
			b.SendMessage(userID, msg)
		} else {
			subsCount, _ := storage.GetUserSubscriptionCount(b.db, userID)
			msg := fmt.Sprintf("👤 Пользователь\nID: %d\nПодписок: %d", userID, subsCount)
			b.SendMessage(userID, msg)
		}

	case txt == "/help":
		if b.IsAdmin(userID) {
			b.SendMessage(userID, "Доступные команды:\n"+
				"/start – информация\n"+
				"/help – список команд\n"+
				"/latest – новости\n"+
				"/mysources – подписки\n"+
				"/autopost – авторассылка\n\n"+
				"👑 Админские:\n"+
				"/addsource – добавить источник\n"+
				"/removesource – удалить источник\n"+
				"/listsources – список источников\n"+
				"/broadcast – рассылка всем\n"+
				"/setchannel <url> – задать ссылку на канал\n"+
				"/setmanual <url> – задать ссылку на инструкцию\n"+
				"/getsettings – показать все настройки")
		} else {
			b.SendMessage(userID, "Доступные команды:\n"+
				"/start – информация\n"+
				"/help – список команд\n"+
				"/latest – новости\n"+
				"/mysources – подписки\n"+
				"/autopost – авторассылка")
		}

	case strings.HasPrefix(txt, "/autopost "):
		parts := strings.Fields(txt)[1:]
		var validTimes []string
		for _, p := range parts {
			if len(p) == 5 && p[2] == ':' {
				validTimes = append(validTimes, p)
			}
		}
		if len(validTimes) > 6 {
			b.SendMessage(userID, "⚠️ Максимум 6 времён")
		} else if len(validTimes) == 0 {
			b.SendMessage(userID, "⚠️ Неверный формат времени")
		} else {
			_ = storage.SetUserAutopost(b.db, userID, validTimes)
			b.SendMessage(userID, "✅ Время авторассылки обновлено: "+strings.Join(validTimes, ", "))
		}

	case txt == "/autopost":
		b.ShowAutopostMenu(userID)

	case txt == "/latest":
		// подгружаем новые новости только по подпискам юзера
		_ = storage.FetchAndStoreNewsForUser(b.db, userID)
		b.latestPage[userID] = 1
		b.ShowLatestNews(userID, nil)

	case txt == "/mysources":
		b.ShowSourcesMenu(userID)

	case txt == "/addsource" && b.IsAdmin(userID):
		b.SendMessage(userID, "Введите URL источника для добавления:")
		b.pending[userID] = "addsource"

	case txt == "/removesource" && b.IsAdmin(userID):
		b.SendMessage(userID, "Введите URL источника для удаления:")
		b.pending[userID] = "removesource"

	case txt == "/listsources" && b.IsAdmin(userID):
		sources := storage.MustGetAllSources(b.db)
		if len(sources) == 0 {
			b.SendMessage(userID, "⚠️ В базе нет источников")
		} else {
			out := "📑 Источники:\n"
			for i, s := range sources {
				out += fmt.Sprintf("%d. %s\n", i+1, s)
			}
			b.SendMessage(userID, out)
		}

	case txt == "/broadcast" && b.IsAdmin(userID):
		b.SendMessage(userID, "Введите текст рассылки:")
		b.pending[userID] = "broadcast"

	// 🔹 Новые команды для settings
	case strings.HasPrefix(txt, "/setchannel ") && b.IsAdmin(userID):
		url := strings.TrimSpace(strings.TrimPrefix(txt, "/setchannel "))
		if url == "" {
			b.SendMessage(userID, "⚠️ Укажите ссылку на канал")
		} else {
			_ = storage.SetSetting(b.db, "channel", url)
			b.SendMessage(userID, "✅ Ссылка на канал обновлена")
		}

	case strings.HasPrefix(txt, "/setmanual ") && b.IsAdmin(userID):
		url := strings.TrimSpace(strings.TrimPrefix(txt, "/setmanual "))
		if url == "" {
			b.SendMessage(userID, "⚠️ Укажите ссылку на инструкцию")
		} else {
			_ = storage.SetSetting(b.db, "manual", url)
			b.SendMessage(userID, "✅ Ссылка на инструкцию обновлена")
		}

	case txt == "/getsettings" && b.IsAdmin(userID):
		settings, _ := storage.GetAllSettings(b.db)
		if len(settings) == 0 {
			b.SendMessage(userID, "⚠️ Настройки пока пустые")
		} else {
			out := "⚙️ Текущие настройки:\n"
			for k, v := range settings {
				out += fmt.Sprintf("%s = %s\n", k, v)
			}
			b.SendMessage(userID, out)
		}

	default:
		log.Printf("Сообщение: %s", txt)
	}
}
