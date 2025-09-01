package bot

import(
	"log"
	"strings"
	"time"
)

type Bot struct {
	Token   string
	APIBase string
	Sent    map[string]bool //cache of sent links
}

func New(token string) *Bot {
	return &Bot{
		Token:   token,
		APIBase: "https://api.telegram.org/bot" + token + "/",
		Sent:    make(map[string]bool),
	}
}

func (b *Bot) Start() {
	log.Println("Bot started ...")

	var offset int
	for {
		updates, err := b.GetUpdates(offset, 30)
		if err != nil {
			if strings.Contains(err.Error(), "Client.Timeout") {
				continue
			}
			log.Printf("getUpdates error: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			if u.Message != nil {
				b.HandleMessage(u.Message)
			}
		}
	}
}
