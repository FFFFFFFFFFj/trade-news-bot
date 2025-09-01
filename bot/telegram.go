package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTP client with timeout to avoid hanging
var client = &http.Client{Timeout: 35 * time.Second}

// Update from Telegram
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

// Message from Telegram
type Message struct {
	MessageID int    `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

type Chat struct {
	ID int64 `json:"id"`
}

// Response to getUpdates
type GetUpdatesResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}


// GetUpdates gets updates
func (b *Bot) GetUpdates(offset int, timeout int) ([]Update, error) {
	url := fmt.Sprintf("%sgetUpdates?timeout=%d&offset=%d", b.APIBase, timeout, offset)
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


// SendMessage sends a message
func (b *Bot) SendMessage(chatID int64, text string) error {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}
	data, _ := json.Marshal(payload)
	resp, err := client.Post(b. APIBase+"sendMessage", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
