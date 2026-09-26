package notify

import "net/url"

func init() {
	Register(Teams{})
}

// Teams - Microsoft Teams 웹후크 (Workflows / Incoming Webhook 공통 Adaptive Card 형식).
type Teams struct{}

type teamsPayload struct {
	Type        string            `json:"type"`
	Attachments []teamsAttachment `json:"attachments"`
}

type teamsAttachment struct {
	ContentType string       `json:"contentType"`
	Content     adaptiveCard `json:"content"`
}

type adaptiveCard struct {
	Schema  string           `json:"$schema"`
	Type    string           `json:"type"`
	Version string           `json:"version"`
	Body    []adaptiveText   `json:"body"`
	Actions []adaptiveAction `json:"actions,omitempty"`
}

type adaptiveText struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Weight string `json:"weight,omitempty"`
	Size   string `json:"size,omitempty"`
	Color  string `json:"color,omitempty"`
	Wrap   bool   `json:"wrap"`
}

type adaptiveAction struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (Teams) Name() string { return "teams" }

func (Teams) Matches(endpoint *url.URL) bool {
	return hostEndsWith(endpoint, ".logic.azure.com", ".powerplatform.com", ".webhook.office.com")
}

func (Teams) Format(e Event, _ *url.URL) (any, error) {
	body := []adaptiveText{{
		Type: "TextBlock", Text: e.Title(), Weight: "Bolder", Size: "Medium", Color: teamsColor(e.Outcome), Wrap: true,
	}}
	for _, line := range e.Lines() {
		body = append(body, adaptiveText{Type: "TextBlock", Text: line, Wrap: true})
	}
	var actions []adaptiveAction
	if e.RunURL != "" {
		actions = []adaptiveAction{{Type: "Action.OpenUrl", Title: "View run", URL: e.RunURL}}
	}
	return teamsPayload{
		Type: "message",
		Attachments: []teamsAttachment{{
			ContentType: "application/vnd.microsoft.card.adaptive",
			Content: adaptiveCard{
				Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
				Type:    "AdaptiveCard",
				Version: "1.4",
				Body:    body,
				Actions: actions,
			},
		}},
	}, nil
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
