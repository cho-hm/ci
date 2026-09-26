package notify

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

func init() {
	Register(Discord{})
}

// Discord - Discord 웹후크. Embed 의 author / 색상 바 / inline field / footer / timestamp 로 표현한다.
type Discord struct{}

// Discord embed 길이 제한
const (
	discordFieldLimit = 1024
	discordTitleLimit = 256
)

type discordPayload struct {
	Username string         `json:"username"`
	Embeds   []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Author    *discordAuthor `json:"author,omitempty"`
	Title     string         `json:"title"`
	URL       string         `json:"url,omitempty"`
	Color     int            `json:"color"`
	Fields    []discordField `json:"fields"`
	Footer    discordFooter  `json:"footer"`
	Timestamp string         `json:"timestamp"`
}

type discordAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordFooter struct {
	Text string `json:"text"`
}

func (Discord) Name() string { return "discord" }

func (Discord) Matches(endpoint *url.URL) bool {
	return hostIs(endpoint, "discord.com", "discordapp.com", "ptb.discord.com", "canary.discord.com") &&
		strings.HasPrefix(endpoint.Path, "/api/webhooks/")
}

func (Discord) Format(m Message, _ *url.URL) (any, error) {
	c := m.Card
	embed := discordEmbed{
		Title:     truncate(c.Title, discordTitleLimit),
		URL:       c.TitleURL,
		Color:     c.Outcome.color(),
		Fields:    discordFields(c),
		Footer:    discordFooter{Text: c.Footer},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if c.Author.Label != "" {
		embed.Author = &discordAuthor{Name: c.Author.Label, URL: c.Author.URL, IconURL: c.AvatarURL}
	}
	return discordPayload{Username: "ci", Embeds: []discordEmbed{embed}}, nil
}

func discordFields(c Card) []discordField {
	var fields []discordField
	for _, f := range c.Facts {
		fields = append(fields, discordField{Name: f.Icon + " " + f.Label, Value: mdLink(f.Value, f.URL), Inline: true})
	}
	if c.Commit != nil {
		fields = append(fields, discordField{Name: c.Labels.Commit, Value: truncate(mdCommit(c.Commit), discordFieldLimit)})
	}
	if len(c.Phases) > 0 {
		lines := make([]string, 0, len(c.Phases))
		for _, p := range c.Phases {
			lines = append(lines, mdPhase(p))
		}
		fields = append(fields, discordField{Name: c.Labels.Phases, Value: truncate(strings.Join(lines, "\n"), discordFieldLimit)})
	}
	if len(c.Artifacts) > 0 {
		fields = append(fields, discordField{Name: c.Labels.Artifacts, Value: codeBlock(strings.Join(c.Artifacts, "\n"), discordFieldLimit)})
	}
	if len(c.Errors) > 0 {
		fields = append(fields, discordField{Name: c.Labels.Error, Value: codeBlock(strings.Join(c.Errors, "\n"), discordFieldLimit)})
	}
	if len(c.Links) > 0 {
		fields = append(fields, discordField{Name: "\U0001F517", Value: mdLinks(c.Links)})
	}
	return fields
}

// ---- Markdown 헬퍼 (Discord, Teams 공용) ----

func mdLink(text, url string) string {
	if url == "" {
		return text
	}
	return fmt.Sprintf("[%s](%s)", text, url)
}

func mdCommit(c *CommitView) string {
	return strings.TrimSpace(mdLink("`"+c.ShortSha+"`", c.URL) + " " + c.Message)
}

func mdPhase(p PhaseView) string {
	parts := []string{p.Icon + " **" + p.Name + "**", p.Status}
	if p.Elapsed != "" {
		parts = append(parts, "`"+p.Elapsed+"`")
	}
	if p.Detail != "" {
		parts = append(parts, "_"+p.Detail+"_")
	}
	return strings.Join(parts, " · ")
}

func mdLinks(links []Link) string {
	parts := make([]string, 0, len(links))
	for _, l := range links {
		parts = append(parts, mdLink(l.Label, l.URL))
	}
	return strings.Join(parts, " · ")
}

// codeBlock - 제한 길이 안에서 ``` 코드 블록으로 감싼다.
func codeBlock(s string, limit int) string {
	const fence = "```"
	return fence + "\n" + truncate(s, limit-len(fence)*2-2) + "\n" + fence
}
