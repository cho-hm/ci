package notify

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

func init() {
	Register(Slack{})
}

// Slack - Slack Incoming Webhook. 색상 바 / author / 2열 field / footer 가 있는 attachment 로 표현한다.
// Block Kit 대신 attachment 를 쓰는 이유: Slack 호환 플랫폼(Mattermost, Rocket.Chat)도 그대로 렌더링하기 때문.
type Slack struct{}

type slackPayload struct {
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments"`
}

type slackAttachment struct {
	Fallback   string       `json:"fallback"`
	Color      string       `json:"color"`
	AuthorName string       `json:"author_name,omitempty"`
	AuthorLink string       `json:"author_link,omitempty"`
	AuthorIcon string       `json:"author_icon,omitempty"`
	Title      string       `json:"title"`
	TitleLink  string       `json:"title_link,omitempty"`
	Fields     []slackField `json:"fields"`
	Footer     string       `json:"footer"`
	Ts         int64        `json:"ts"`
	MrkdwnIn   []string     `json:"mrkdwn_in"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func (Slack) Name() string { return "slack" }

func (Slack) Matches(endpoint *url.URL) bool {
	return hostIs(endpoint, "hooks.slack.com")
}

func (Slack) Format(m Message, _ *url.URL) (any, error) {
	c := m.Card
	return slackPayload{
		Text: c.Title,
		Attachments: []slackAttachment{{
			Fallback:   c.Summary(),
			Color:      c.Outcome.hexColor(),
			AuthorName: c.Author.Label,
			AuthorLink: c.Author.URL,
			AuthorIcon: c.AvatarURL,
			Title:      c.Title,
			TitleLink:  c.TitleURL,
			Fields:     slackFields(c),
			Footer:     c.Footer,
			Ts:         time.Now().Unix(),
			MrkdwnIn:   []string{"fields"},
		}},
	}, nil
}

func slackFields(c Card) []slackField {
	var fields []slackField
	for _, f := range c.Facts {
		fields = append(fields, slackField{Title: f.Icon + " " + f.Label, Value: slackLink(f.Value, f.URL), Short: true})
	}
	if c.Commit != nil {
		sha := slackLink("`"+c.Commit.ShortSha+"`", c.Commit.URL)
		fields = append(fields, slackField{Title: c.Labels.Commit, Value: strings.TrimSpace(sha + " " + slackEscape(c.Commit.Message))})
	}
	if len(c.Phases) > 0 {
		lines := make([]string, 0, len(c.Phases))
		for _, p := range c.Phases {
			lines = append(lines, slackPhase(p))
		}
		fields = append(fields, slackField{Title: c.Labels.Phases, Value: strings.Join(lines, "\n")})
	}
	if len(c.Artifacts) > 0 {
		fields = append(fields, slackField{Title: c.Labels.Artifacts, Value: "```" + strings.Join(c.Artifacts, "\n") + "```"})
	}
	if len(c.Errors) > 0 {
		fields = append(fields, slackField{Title: c.Labels.Error, Value: "```" + slackEscape(truncate(strings.Join(c.Errors, "\n"), 1500)) + "```"})
	}
	if len(c.Links) > 0 {
		parts := make([]string, 0, len(c.Links))
		for _, l := range c.Links {
			parts = append(parts, slackLink(l.Label, l.URL))
		}
		fields = append(fields, slackField{Title: "\U0001F517", Value: strings.Join(parts, "  ·  ")})
	}
	return fields
}

func slackPhase(p PhaseView) string {
	parts := []string{p.Icon + " *" + p.Name + "*", p.Status}
	if p.Elapsed != "" {
		parts = append(parts, "`"+slackEscape(p.Elapsed)+"`")
	}
	if p.Detail != "" {
		parts = append(parts, "_"+slackEscape(p.Detail)+"_")
	}
	return strings.Join(parts, " · ")
}

func slackLink(text, url string) string {
	if url == "" {
		return slackEscape(text)
	}
	return fmt.Sprintf("<%s|%s>", url, slackEscape(text))
}

// slackEscape - Slack mrkdwn 제어 문자(&, <, >) 이스케이프.
func slackEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
