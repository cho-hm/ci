package notify

import (
	"ci/core/constant"
	"fmt"
	"strings"
	"time"
)

// Card - 플랫폼 공통 표현 모델 (Presentation Model).
// Event 를 사람이 읽는 문구·아이콘·링크로 한 번만 해석해 두고, 각 Notifier 는 이것을 자기 플랫폼의 꾸밈 요소로 렌더링한다.
type Card struct {
	Outcome   Outcome
	Title     string
	TitleURL  string
	Author    Link   // 실행자 (이름 + 프로필 링크)
	AvatarURL string // 실행자 프로필 이미지
	Subtitle  string // 예: "by someone"
	Facts     []Fact
	Commit    *CommitView
	Phases    []PhaseView
	Artifacts []string
	Errors    []string
	Links     []Link
	Footer    string
	Labels    SectionLabels
}

type Fact struct {
	Icon  string
	Label string
	Value string
	URL   string
}

type Link struct {
	Label string
	URL   string
}

type CommitView struct {
	ShortSha string
	Message  string
	URL      string
}

type PhaseView struct {
	Icon    string
	Name    string
	Status  string
	Elapsed string // 스킵 등으로 측정값이 없으면 빈 문자열
	Detail  string // 스킵 사유
}

// SectionLabels - 아이콘이 포함된 섹션 제목.
type SectionLabels struct {
	Commit, Phases, Artifacts, Error string
}

func NewCard(e Event, t texts) Card {
	return Card{
		Outcome:   e.Outcome,
		Title:     title(e, t),
		TitleURL:  e.RunURL,
		Author:    Link{Label: e.Actor, URL: e.ActorURL()},
		AvatarURL: e.ActorAvatarURL(),
		Subtitle:  subtitle(e, t),
		Facts:     facts(e, t),
		Commit:    commit(e),
		Phases:    phases(e, t),
		Artifacts: e.Artifacts(),
		Errors:    e.FailureCauses(),
		Links:     links(e, t),
		Footer:    footer(e),
		Labels: SectionLabels{
			Commit:    "\U0001F9E9 " + t.Commit,
			Phases:    "\U0001F4CA " + t.Phases,
			Artifacts: "\U0001F433 " + t.Artifacts,
			Error:     "\U0001F525 " + t.Error,
		},
	}
}

// Line - "📦 Repository: owner/repo" 형태의 평문 표현.
func (f Fact) Line() string {
	return fmt.Sprintf("%s %s: %s", f.Icon, f.Label, f.Value)
}

// Line - "✅ build · succeeded · 2m 21s · reason" 형태의 평문 표현.
func (p PhaseView) Line() string {
	parts := []string{p.Icon + " " + p.Name, p.Status}
	if p.Elapsed != "" {
		parts = append(parts, p.Elapsed)
	}
	if p.Detail != "" {
		parts = append(parts, p.Detail)
	}
	return strings.Join(parts, " · ")
}

// Summary - 링크/서식 없는 한 덩어리 평문 (미리보기, 폴백 텍스트용).
func (c Card) Summary() string {
	lines := []string{c.Title}
	for _, f := range c.Facts {
		lines = append(lines, f.Line())
	}
	return strings.Join(lines, "\n")
}

func title(e Event, t texts) string {
	workflow := e.Workflow
	if workflow == "" {
		workflow = "CI"
	}
	status := map[Outcome]string{Running: t.Started, Succeeded: t.Succeeded, Failed: t.Failed}[e.Outcome]
	s := fmt.Sprintf("%s %s %s · %s", e.Outcome.icon(), workflow, status, e.Repository)
	if e.RunNumber != "" {
		s += " #" + e.RunNumber
	}
	if e.RunAttempt != "" && e.RunAttempt != "1" {
		s += " (" + fmt.Sprintf(t.Attempt, e.RunAttempt) + ")"
	}
	return s
}

func subtitle(e Event, t texts) string {
	if e.Actor == "" {
		return ""
	}
	return fmt.Sprintf(t.By, e.Actor)
}

func facts(e Event, t texts) []Fact {
	candidates := []Fact{
		{Icon: "\U0001F4E6", Label: t.Repository, Value: e.Repository, URL: e.RepositoryURL()},
		{Icon: "\U0001F33F", Label: t.Ref, Value: e.RefName, URL: e.RefURL()},
		{Icon: "\U0001F516", Label: t.Trigger, Value: joinNonEmpty(" · ", e.EventName, e.RefType)},
		{Icon: "\U0001F6E0", Label: t.Env, Value: e.Env},
	}
	if e.Kind == Started {
		candidates = append(candidates, Fact{Icon: "\U0001F4CB", Label: t.Tasks, Value: strings.Join(e.Tasks, ", ")})
	} else {
		candidates = append(candidates, Fact{Icon: "⏱", Label: t.Duration, Value: formatDuration(e.Elapsed)})
	}
	var result []Fact
	for _, f := range candidates {
		if f.Value != "" {
			result = append(result, f)
		}
	}
	return result
}

func commit(e Event) *CommitView {
	if e.Sha == "" {
		return nil
	}
	return &CommitView{ShortSha: shortSha(e.Sha), Message: e.CommitMessage, URL: e.CommitURL()}
}

func phases(e Event, t texts) []PhaseView {
	views := make([]PhaseView, 0, len(e.Phases))
	for _, p := range e.Phases {
		view := PhaseView{Name: p.Phase}
		switch p.Status {
		case constant.PhaseSuccess:
			view.Icon, view.Status = "✅", t.PhaseSucceeded
		case constant.PhaseFailure:
			view.Icon, view.Status = "❌", t.PhaseFailed
		default:
			view.Icon, view.Status, view.Detail = "⏭", t.PhaseSkipped, p.Reason
		}
		if p.Elapsed > 0 {
			view.Elapsed = formatDuration(p.Elapsed)
		}
		views = append(views, view)
	}
	return views
}

func links(e Event, t texts) []Link {
	var result []Link
	if e.RunURL != "" {
		result = append(result, Link{Label: t.ViewRun, URL: e.RunURL})
	}
	if url := e.CommitURL(); url != "" {
		result = append(result, Link{Label: t.ViewCommit, URL: url})
	}
	return result
}

func footer(e Event) string {
	version := e.Version
	if version == "" {
		version = "dev"
	}
	return joinNonEmpty(" · ", "cho-hm/ci "+version, e.Runner)
}

func (o Outcome) icon() string {
	switch o {
	case Succeeded:
		return "✅"
	case Failed:
		return "❌"
	default:
		return "\U0001F680"
	}
}

// color - 상태별 대표 색상 (RGB).
func (o Outcome) color() int {
	switch o {
	case Succeeded:
		return 0x22C55E
	case Failed:
		return 0xEF4444
	default:
		return 0x3B82F6
	}
}

func (o Outcome) hexColor() string {
	return fmt.Sprintf("#%06X", o.color())
}

func phaseStatusName(s constant.PhaseStatus) string {
	switch s {
	case constant.PhaseSuccess:
		return "success"
	case constant.PhaseFailure:
		return "failure"
	default:
		return "skipped"
	}
}

// formatDuration - "3m 12s" 형태. 1초 미만은 "<1s".
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Second {
		return "<1s"
	}
	h, m, s := int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60
	var parts []string
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if s > 0 {
		parts = append(parts, fmt.Sprintf("%ds", s))
	}
	return strings.Join(parts, " ")
}

func shortSha(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func joinNonEmpty(sep string, values ...string) string {
	var parts []string
	for _, v := range values {
		if v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, sep)
}

// truncate - 플랫폼 길이 제한을 넘지 않도록 rune 단위로 자른다.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
