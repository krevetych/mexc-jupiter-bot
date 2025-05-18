package tg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type tgMsg struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

func SendMessage(botToken string, chatID int64, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	msg := tgMsg{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	res, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api returned status %s", res.Status)
	}

	return nil
}
