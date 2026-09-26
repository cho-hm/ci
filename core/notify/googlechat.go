package notify

import (
	"fmt"
	"html"
	"net/url"
	"strings"
)

func init() {
	Register(GoogleChat{})
}

// GoogleChat - Google Chat 스페이스 웹후크 (cardsV2).
// 프로필 이미지 헤더, 라벨이 달린 decoratedText, 섹션 제목, 색상 글자, 버튼으로 표현한다.
type GoogleChat struct{}

type gchat = map[string]any

func (GoogleChat) Name() string { return "googlechat" }

func (GoogleChat) Matches(endpoint *url.URL) bool {
	return hostIs(endpoint, "chat.googleapis.com")
}

func (GoogleChat) Format(m Message, _ *url.URL) (any, error) {
	c := m.Card
	header := gchat{"title": c.Title}
	if c.Subtitle != "" {
		header["subtitle"] = c.Subtitle
	}
	if c.AvatarURL != "" {
		header["imageUrl"] = c.AvatarURL
		header["imageType"] = "CIRCLE"
	}
	return gchat{
		"cardsV2": []gchat{{
			"cardId": "ci-notify",
			"card":   gchat{"header": header, "sections": gchatSections(c)},
		}},
	}, nil
}

func gchatSections(c Card) []gchat {
	var sections []gchat
	if len(c.Facts) > 0 {
		widgets := make([]gchat, 0, len(c.Facts))
		for _, f := range c.Facts {
			widgets = append(widgets, gchat{"decoratedText": gchat{
				"topLabel": f.Icon + " " + f.Label,
				"text":     htmlLink(f.Value, f.URL),
			}})
		}
		sections = append(sections, gchat{"widgets": widgets})
	}
	if c.Commit != nil {
		sections = append(sections, gchatTextSection(c.Labels.Commit, gchatCommit(c.Commit)))
	}
	if len(c.Phases) > 0 {
		lines := make([]string, 0, len(c.Phases))
		for _, p := range c.Phases {
			lines = append(lines, htmlPhase(p))
		}
		sections = append(sections, gchatTextSection(c.Labels.Phases, strings.Join(lines, "<br>")))
	}
	if len(c.Artifacts) > 0 {
		sections = append(sections, gchatTextSection(c.Labels.Artifacts, gchatLines(c.Artifacts)))
	}
	if len(c.Errors) > 0 {
		text := fmt.Sprintf(`<font color="%s">%s</font>`, Failed.hexColor(), gchatLines([]string{truncate(strings.Join(c.Errors, "\n"), 2000)}))
		sections = append(sections, gchatTextSection(c.Labels.Error, text))
	}
	if len(c.Links) > 0 {
		buttons := make([]gchat, 0, len(c.Links))
		for _, l := range c.Links {
			buttons = append(buttons, gchat{"text": l.Label, "onClick": gchat{"openLink": gchat{"url": l.URL}}})
		}
		sections = append(sections, gchat{"widgets": []gchat{{"buttonList": gchat{"buttons": buttons}}}})
	}
	footer := fmt.Sprintf(`<font color="#808080">%s</font>`, html.EscapeString(c.Footer))
	return append(sections, gchat{"widgets": []gchat{{"textParagraph": gchat{"text": footer}}}})
}

func gchatTextSection(header, text string) gchat {
	return gchat{"header": header, "widgets": []gchat{{"textParagraph": gchat{"text": text}}}}
}

// gchatCommit - Google Chat 카드는 <code> 를 지원하지 않아 굵은 글씨로 sha 를 강조한다.
func gchatCommit(c *CommitView) string {
	sha := "<b>" + html.EscapeString(c.ShortSha) + "</b>"
	if c.URL != "" {
		sha = fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(c.URL), sha)
	}
	return strings.TrimSpace(sha + " " + html.EscapeString(c.Message))
}

// gchatLines - 줄바꿈은 <br> 로만 표현된다.
func gchatLines(lines []string) string {
	return strings.ReplaceAll(html.EscapeString(strings.Join(lines, "\n")), "\n", "<br>")
}

// ---- HTML 헬퍼 (Google Chat, Telegram 공용) ----

func htmlLink(text, url string) string {
	if url == "" {
		return html.EscapeString(text)
	}
	return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(url), html.EscapeString(text))
}

func htmlCommit(c *CommitView) string {
	sha := "<code>" + html.EscapeString(c.ShortSha) + "</code>"
	if c.URL != "" {
		sha = fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(c.URL), sha)
	}
	return strings.TrimSpace(sha + " " + html.EscapeString(c.Message))
}

func htmlPhase(p PhaseView) string {
	parts := []string{p.Icon + " <b>" + html.EscapeString(p.Name) + "</b>", html.EscapeString(p.Status)}
	if p.Elapsed != "" {
		parts = append(parts, html.EscapeString(p.Elapsed))
	}
	if p.Detail != "" {
		parts = append(parts, "<i>"+html.EscapeString(p.Detail)+"</i>")
	}
	return strings.Join(parts, " · ")
}
