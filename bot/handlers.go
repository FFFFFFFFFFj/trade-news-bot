package bot

import (
	"log"
	"strings"
)

func (b *Bot) HandleMessage(m *Message) {
	txt := strings.TrimSpace(m.Text)
	switch txt {
	case "/start":
		b.SendMessage(m.Chat.ID, "Hi, I'm bot. Write /latest")
	case "/latest":
		b.SendMessage(m.Chat.ID, "This is news")
	default:
		log.Printf("Got message: %s", txt)
	}
}
