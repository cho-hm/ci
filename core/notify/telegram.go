package notify

import (
	"errors"
	"html"
	"net/url"
	"strings"
)

func init() {
	Register(Telegram{})
}

// Telegram - Telegram Bot API sendMessage (HTML parse mode).
// 굵은 제목, 링크, 코드/pre 블록, 이탤릭 footer 로 표현한다.
// 웹후크 URL 형태: https://api.telegram.org/bot<TOKEN>/sendMessage?chat_id=<CHAT_ID>
type Telegram struct{}

const telegramTextLimit = 4096

type telegramPayload struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

func (Telegram) Name() string { return "telegram" }

func (Telegram) Matches(endpoint *url.URL) bool {
	return hostIs(endpoint, "api.telegram.org")
}

func (Telegram) Format(m Message, endpoint *url.URL) (any, error) {
	chatID := endpoint.Query().Get("chat_id")
	if chatID == "" {
		return nil, errors.New("telegram webhook url requires chat_id query parameter")
	}
	return telegramPayload{
		ChatID:                chatID,
		Text:                  telegramText(m.Card),
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	}, nil
}

func telegramText(c Card) string {
	blocks := []string{telegramHeader(c)}
	if len(c.Facts) > 0 {
		lines := make([]string, 0, len(c.Facts))
		for _, f := range c.Facts {
			lines = append(lines, f.Icon+" <b>"+html.EscapeString(f.Label)+"</b>: "+htmlLink(f.Value, f.URL))
		}
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	if c.Commit != nil {
		blocks = append(blocks, "<b>"+c.Labels.Commit+"</b>\n"+htmlCommit(c.Commit))
	}
	if len(c.Phases) > 0 {
		lines := make([]string, 0, len(c.Phases))
		for _, p := range c.Phases {
			lines = append(lines, htmlPhase(p))
		}
		blocks = append(blocks, "<b>"+c.Labels.Phases+"</b>\n"+strings.Join(lines, "\n"))
	}
	if len(c.Artifacts) > 0 {
		blocks = append(blocks, "<b>"+c.Labels.Artifacts+"</b>\n<pre>"+html.EscapeString(strings.Join(c.Artifacts, "\n"))+"</pre>")
	}
	if len(c.Errors) > 0 {
		errText := truncate(strings.Join(c.Errors, "\n"), 1500)
		blocks = append(blocks, "<b>"+c.Labels.Error+"</b>\n<pre>"+html.EscapeString(errText)+"</pre>")
	}
	if len(c.Links) > 0 {
		parts := make([]string, 0, len(c.Links))
		for _, l := range c.Links {
			parts = append(parts, htmlLink(l.Label, l.URL))
		}
		blocks = append(blocks, "\U0001F517 "+strings.Join(parts, " · "))
	}
	blocks = append(blocks, "<i>"+html.EscapeString(c.Footer)+"</i>")
	text := strings.Join(blocks, "\n\n")
	if len([]rune(text)) > telegramTextLimit {
		// HTML 태그가 잘리지 않도록 서식 없는 요약으로 대체한다.
		return html.EscapeString(truncate(c.Summary(), telegramTextLimit))
	}
	return text
}

func telegramHeader(c Card) string {
	header := "<b>" + html.EscapeString(c.Title) + "</b>"
	if c.Author.Label != "" {
		header += "\n\U0001F464 " + htmlLink(c.Author.Label, c.Author.URL)
	}
	return header
}
