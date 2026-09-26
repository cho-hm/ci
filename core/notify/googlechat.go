package notify

import (
	"fmt"
	"net/url"
	"strings"
)

func init() {
	Register(GoogleChat{})
}

// GoogleChat - Google Chat 스페이스 웹후크 (텍스트 메시지 형식).
type GoogleChat struct{}

type googleChatPayload struct {
	Text string `json:"text"`
}

func (GoogleChat) Name() string { return "googlechat" }

func (GoogleChat) Matches(endpoint *url.URL) bool {
	return hostIs(endpoint, "chat.googleapis.com")
}

func (GoogleChat) Format(e Event, _ *url.URL) (any, error) {
	parts := []string{"*" + e.Title() + "*", strings.Join(e.Lines(), "\n")}
	if e.RunURL != "" {
		parts = append(parts, fmt.Sprintf("<%s|View run>", e.RunURL))
	}
	return googleChatPayload{Text: strings.Join(parts, "\n")}, nil
}
