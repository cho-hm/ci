package notify

import (
	"net/url"
	"strings"
	"time"
)

func init() {
	Register(Discord{})
}

// Discord - Discord 웹후크 (embed 형식).
type Discord struct{}

type discordPayload struct {
	Username string         `json:"username"`
	Embeds   []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string `json:"title"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description"`
	Color       int    `json:"color"`
	Timestamp   string `json:"timestamp"`
}

func (Discord) Name() string { return "discord" }

func (Discord) Matches(endpoint *url.URL) bool {
	return hostIs(endpoint, "discord.com", "discordapp.com", "ptb.discord.com", "canary.discord.com") &&
		strings.HasPrefix(endpoint.Path, "/api/webhooks/")
}

func (Discord) Format(e Event, _ *url.URL) (any, error) {
	return discordPayload{
		Username: "ci",
		Embeds: []discordEmbed{{
			Title:       e.Title(),
			URL:         e.RunURL,
			Description: strings.Join(e.Lines(), "\n"),
			Color:       e.Outcome.color(),
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}},
	}, nil
}
