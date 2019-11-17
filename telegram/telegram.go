package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

const (
	telegramChatIDEnvVar   = "TELEGRAM_CHAT_ID"
	telegramApiTokenEnvVar = "TELEGRAM_API_TOKEN"
)

type TelegramClient struct {
	telegramChatID string
	telegramApiURL string
}

func NewTelegramClient() *TelegramClient {
	tc := TelegramClient{}
	tc.telegramChatID = os.Getenv(telegramChatIDEnvVar)
	apiToken := os.Getenv(telegramApiTokenEnvVar)
	tc.telegramApiURL = fmt.Sprintf("https://api.telegram.org/bot%v/sendMessage", apiToken)
	return &tc
}

func (tc *TelegramClient) SendMessage(text string) {
	body := request{
		ChatId: tc.telegramChatID,
		Text:   text,
	}
	jsonStr, err := json.Marshal(body)
	if err != nil {
		log.WithField("text", text).WithError(err).Error("Failed to marshal json")
		return
	}

	resp, err := http.Post(tc.telegramApiURL, "application/json", bytes.NewBuffer(jsonStr))
	if err != nil {
		log.WithError(err).Error("Failed to send POST to telegram")
		return
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Error("Failed to read telegram response")
		return
	}
	r := response{}
	json.Unmarshal(respBody, &r)

	if !r.Ok {
		log.WithField("text", text).Error("Failed to send to telegrams")
	}
}

type request struct {
	ChatId string `json:"chat_id"`
	Text   string `json:"text"`
}

type response struct {
	Ok bool `json:"ok"`
}
