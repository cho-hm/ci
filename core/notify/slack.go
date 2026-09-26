package notify

import (
	"net/url"
	"strings"
)

func init() {
	Register(Slack{})
}

// Slack - Slack Incoming Webhook. Slack 호환 플랫폼(Mattermost, Rocket.Chat)도 이 형식을 받는다.
type Slack struct{}

type slackPayload struct {
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments"`
}

type slackAttachment struct {
	Color     string `json:"color"`
	Title     string `json:"title"`
	TitleLink string `json:"title_link,omitempty"`
	Text      string `json:"text"`
}

func (Slack) Name() string { return "slack" }

func (Slack) Matches(endpoint *url.URL) bool {
	return hostIs(endpoint, "hooks.slack.com")
}

func (Slack) Format(e Event, _ *url.URL) (any, error) {
	return slackPayload{
		Text: e.Title(),
		Attachments: []slackAttachment{{
			Color:     e.Outcome.hexColor(),
			Title:     e.Title(),
			TitleLink: e.RunURL,
			Text:      strings.Join(e.Lines(), "\n"),
		}},
	}, nil
}
