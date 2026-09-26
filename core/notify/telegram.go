package notify

import (
	"errors"
	"net/url"
	"strings"
)

func init() {
	Register(Telegram{})
}

// Telegram - Telegram Bot API sendMessage.
// 웹후크 URL 형태: https://api.telegram.org/bot<TOKEN>/sendMessage?chat_id=<CHAT_ID>
type Telegram struct{}

type telegramPayload struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

func (Telegram) Name() string { return "telegram" }

func (Telegram) Matches(endpoint *url.URL) bool {
	return hostIs(endpoint, "api.telegram.org")
}

func (Telegram) Format(e Event, endpoint *url.URL) (any, error) {
	chatID := endpoint.Query().Get("chat_id")
	if chatID == "" {
		return nil, errors.New("telegram webhook url requires chat_id query parameter")
	}
	lines := append([]string{e.Title()}, e.Lines()...)
	if e.RunURL != "" {
		lines = append(lines, e.RunURL)
	}
	return telegramPayload{
		ChatID:                chatID,
		Text:                  strings.Join(lines, "\n"),
		DisableWebPagePreview: true,
	}, nil
}
