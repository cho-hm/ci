package notify

import (
	"net/url"
	"strings"
)

func init() {
	Register(Teams{})
}

// Teams - Microsoft Teams 웹후크 (Workflows / Incoming Webhook 공통 Adaptive Card).
// 프로필 이미지 + 제목 헤더, FactSet(키-값 표), 섹션별 TextBlock, 상태 색상, 링크 버튼으로 표현한다.
type Teams struct{}

// adaptive - Adaptive Card 요소. 스키마가 깊고 요소 종류가 다양해 map 으로 구성한다.
type adaptive = map[string]any

func (Teams) Name() string { return "teams" }

func (Teams) Matches(endpoint *url.URL) bool {
	return hostEndsWith(endpoint, ".logic.azure.com", ".powerplatform.com", ".webhook.office.com")
}

func (Teams) Format(m Message, _ *url.URL) (any, error) {
	c := m.Card
	card := adaptive{
		"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
		"type":    "AdaptiveCard",
		"version": "1.4",
		"msteams": adaptive{"width": "Full"},
		"body":    teamsBody(c),
	}
	if actions := teamsActions(c); len(actions) > 0 {
		card["actions"] = actions
	}
	return adaptive{
		"type": "message",
		"attachments": []adaptive{{
			"contentType": "application/vnd.microsoft.card.adaptive",
			"content":     card,
		}},
	}, nil
}

func teamsBody(c Card) []adaptive {
	body := []adaptive{teamsHeader(c)}
	if len(c.Facts) > 0 {
		facts := make([]adaptive, 0, len(c.Facts))
		for _, f := range c.Facts {
			facts = append(facts, adaptive{"title": f.Icon + " " + f.Label, "value": mdLink(f.Value, f.URL)})
		}
		body = append(body, adaptive{"type": "FactSet", "facts": facts, "spacing": "Medium"})
	}
	if c.Commit != nil {
		body = append(body, teamsSection(c.Labels.Commit, textBlock(mdCommit(c.Commit)))...)
	}
	if len(c.Phases) > 0 {
		lines := make([]adaptive, 0, len(c.Phases))
		for _, p := range c.Phases {
			lines = append(lines, textBlock(mdPhase(p)))
		}
		body = append(body, teamsSection(c.Labels.Phases, lines...)...)
	}
	if len(c.Artifacts) > 0 {
		block := textBlock(strings.Join(c.Artifacts, "\n\n"))
		block["fontType"] = "Monospace"
		block["size"] = "Small"
		body = append(body, teamsSection(c.Labels.Artifacts, block)...)
	}
	if len(c.Errors) > 0 {
		block := textBlock(truncate(strings.Join(c.Errors, "\n\n"), 2000))
		block["fontType"] = "Monospace"
		block["color"] = "Attention"
		body = append(body, teamsSection(c.Labels.Error, block)...)
	}
	footer := textBlock(c.Footer)
	footer["size"] = "Small"
	footer["isSubtle"] = true
	footer["separator"] = true
	footer["spacing"] = "Medium"
	return append(body, footer)
}

// teamsHeader - [프로필 이미지] | 제목(상태 색상) + 부제.
func teamsHeader(c Card) adaptive {
	titleBlock := textBlock(c.Title)
	titleBlock["weight"] = "Bolder"
	titleBlock["size"] = "Medium"
	titleBlock["color"] = teamsColor(c.Outcome)
	items := []adaptive{titleBlock}
	if c.Subtitle != "" {
		sub := textBlock(c.Subtitle)
		sub["isSubtle"] = true
		sub["spacing"] = "None"
		items = append(items, sub)
	}
	columns := []adaptive{}
	if c.AvatarURL != "" {
		columns = append(columns, adaptive{
			"type": "Column", "width": "auto", "verticalContentAlignment": "Center",
			"items": []adaptive{{"type": "Image", "url": c.AvatarURL, "size": "Small", "style": "Person"}},
		})
	}
	columns = append(columns, adaptive{"type": "Column", "width": "stretch", "verticalContentAlignment": "Center", "items": items})
	return adaptive{"type": "ColumnSet", "columns": columns}
}

func teamsSection(label string, blocks ...adaptive) []adaptive {
	heading := textBlock(label)
	heading["weight"] = "Bolder"
	heading["spacing"] = "Medium"
	return append([]adaptive{heading}, blocks...)
}

func teamsActions(c Card) []adaptive {
	actions := make([]adaptive, 0, len(c.Links))
	for _, l := range c.Links {
		actions = append(actions, adaptive{"type": "Action.OpenUrl", "title": l.Label, "url": l.URL})
	}
	return actions
}

func textBlock(text string) adaptive {
	return adaptive{"type": "TextBlock", "text": text, "wrap": true, "spacing": "Small"}
}

// teamsColor - Adaptive Card 는 임의 색상이 아닌 의미 기반 색상만 지원한다.
func teamsColor(o Outcome) string {
	switch o {
	case Succeeded:
		return "Good"
	case Failed:
		return "Attention"
	default:
		return "Accent"
	}
}
