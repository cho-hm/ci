package notify

import (
	"ci/core/constant"
	"net/url"
	"time"
)

func init() {
	Register(JSON{})
}

// JSON - 플랫폼 중립 형식. 전용 Notifier 가 없는 수신측(사내 봇, n8n, Zapier 등)이 직접 해석한다.
// 필드 구성은 외부 계약이므로 호환되지 않는 변경 시 Version 을 올린다.
type JSON struct{}

type jsonPayload struct {
	Version    int         `json:"version"`
	Event      string      `json:"event"`
	Status     string      `json:"status"`
	Title      string      `json:"title"`
	Workflow   string      `json:"workflow"`
	Repository string      `json:"repository"`
	Ref        string      `json:"ref"`
	Sha        string      `json:"sha"`
	Actor      string      `json:"actor"`
	RunURL     string      `json:"runUrl"`
	ElapsedSec int64       `json:"elapsedSeconds"`
	Phases     []jsonPhase `json:"phases"`
	Error      string      `json:"error,omitempty"`
	Timestamp  string      `json:"timestamp"`
}

type jsonPhase struct {
	Phase  string `json:"phase"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func (JSON) Name() string { return "json" }

// Matches - 폴백 전용이므로 URL 로는 선택되지 않는다. WEBHOOK_TYPE=json 으로 명시 가능.
func (JSON) Matches(*url.URL) bool { return false }

func (JSON) Format(e Event, _ *url.URL) (any, error) {
	phases := make([]jsonPhase, 0, len(e.Phases))
	for _, p := range e.Phases {
		phases = append(phases, jsonPhase{Phase: p.Phase, Status: phaseStatusName(p.Status), Detail: phaseDetail(p)})
	}
	payload := jsonPayload{
		Version:    1,
		Event:      e.Kind.String(),
		Status:     e.Outcome.String(),
		Title:      e.Title(),
		Workflow:   e.Workflow,
		Repository: e.Repository,
		Ref:        e.RefName,
		Sha:        e.Sha,
		Actor:      e.Actor,
		RunURL:     e.RunURL,
		ElapsedSec: int64(e.Elapsed / time.Second),
		Phases:     phases,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	if e.Cause != nil {
		payload.Error = e.Cause.Error()
	}
	return payload, nil
}

func phaseDetail(p constant.PhaseResult) string {
	if p.Cause != nil {
		return p.Cause.Error()
	}
	return p.Reason
}
