package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//HTTP client whis timeout for making requests to Telegram API
var client = &http.Client{Timeout: 30 * time.Second}

//Base URL for calls to Telegram BotAPI, generated based on the token
var apiBase string

//Update represents a single update from Telegram, which contains the update ID and message
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

//Message represents a message in Telegram, including ID, chat and message text
type Message struct {
	MessageID int     `json:"message_id"`
	Chat      Chat    `json:"chat"`
	Text      string  `json:"text"`
}

//Chat contains data about the chat where the communication takes place
type Chat struct {
	ID int64 `json:"id"`
}

//GetUpdatesResponse describes the Telegram API response to a request for updates
type GetUpdatesResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

func main() {
	//Get the bot token from the environment variable
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN not set")
	}

	//Generate a base URL for API requests
	apiBase = "https://api.telegram.org/bot" + token + "/"
	log.Println("Bot started...")

	var offset int
	//Infinite loop of receiving updates and processing messages
	for{
		updates, err := getUpdates(offset, 30)
		if err != nil {
			log.Printf("getUpdates error: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID +1
			if u.Message != nil {
				handleMessage(u.Message)
			}
		}
	}
}

//getUpdates sends a request to Telegram to get new updates
func getUpdates(offset int, timeout int) ([]Update, error) {
	url := fmt.Sprintf("%sgetUpdates?timeout=%d&offset=%d", apiBase, timeout, offset)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res GetUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if !res.Ok {
		return nil, fmt.Errorf("telegram returned not ok")
	}
	return res.Result, nil
}

//sendMessage sends a text message to the specified chat
func sendMessage(chatID int64, text string) error {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}
	data, _  := json.Marshal(payload)
	resp, err := client.Post(apiBase+"sendMessage", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	//Reset the response body to free up connection resourcees
	io.Copy(io.Discard, resp.Body)
	return nil
}

//handleMessage processes incoming messages and responds to commands
func handleMessage(m *Message) {
	txt := strings.TrimSpace(m.Text)
	switch txt {
	case "/start":
		sendMessage(m.Chat.ID, "Hi, i'm bot, write /latest")
	case "/latest":
		sendMessage(m.Chat.ID, "Whis is news")
	default:
		log.Printf("Got message: %s", txt)
	}
}
