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
//   - v2: refType, eventName, commitMessage, commitUrl, runNumber, runAttempt, env, tasks, runner,
//     ciVersion, phases[].elapsedSeconds, phases[].artifacts 추가
type JSON struct{}

const jsonPayloadVersion = 2

type jsonPayload struct {
	Version       int         `json:"version"`
	Event         string      `json:"event"`
	Status        string      `json:"status"`
	Title         string      `json:"title"`
	Workflow      string      `json:"workflow"`
	Repository    string      `json:"repository"`
	Ref           string      `json:"ref"`
	RefType       string      `json:"refType"`
	EventName     string      `json:"eventName"`
	Sha           string      `json:"sha"`
	CommitMessage string      `json:"commitMessage"`
	CommitURL     string      `json:"commitUrl"`
	Actor         string      `json:"actor"`
	RunURL        string      `json:"runUrl"`
	RunNumber     string      `json:"runNumber"`
	RunAttempt    string      `json:"runAttempt"`
	Env           string      `json:"env"`
	Tasks         []string    `json:"tasks"`
	Runner        string      `json:"runner"`
	CiVersion     string      `json:"ciVersion"`
	ElapsedSec    int64       `json:"elapsedSeconds"`
	Phases        []jsonPhase `json:"phases"`
	Error         string      `json:"error,omitempty"`
	Timestamp     string      `json:"timestamp"`
}

type jsonPhase struct {
	Phase      string   `json:"phase"`
	Status     string   `json:"status"`
	Detail     string   `json:"detail,omitempty"`
	ElapsedSec int64    `json:"elapsedSeconds"`
	Artifacts  []string `json:"artifacts,omitempty"`
}

func (JSON) Name() string { return "json" }

// Matches - 폴백 전용이므로 URL 로는 선택되지 않는다. WEBHOOK_TYPE=json 으로 명시 가능.
func (JSON) Matches(*url.URL) bool { return false }

func (JSON) Format(m Message, _ *url.URL) (any, error) {
	e := m.Event
	phases := make([]jsonPhase, 0, len(e.Phases))
	for _, p := range e.Phases {
		phases = append(phases, jsonPhase{
			Phase:      p.Phase,
			Status:     phaseStatusName(p.Status),
			Detail:     phaseDetail(p),
			ElapsedSec: int64(p.Elapsed / time.Second),
			Artifacts:  p.Artifacts,
		})
	}
	tasks := e.Tasks
	if tasks == nil {
		tasks = []string{}
	}
	payload := jsonPayload{
		Version:       jsonPayloadVersion,
		Event:         e.Kind.String(),
		Status:        e.Outcome.String(),
		Title:         m.Card.Title,
		Workflow:      e.Workflow,
		Repository:    e.Repository,
		Ref:           e.RefName,
		RefType:       e.RefType,
		EventName:     e.EventName,
		Sha:           e.Sha,
		CommitMessage: e.CommitMessage,
		CommitURL:     e.CommitURL(),
		Actor:         e.Actor,
		RunURL:        e.RunURL,
		RunNumber:     e.RunNumber,
		RunAttempt:    e.RunAttempt,
		Env:           e.Env,
		Tasks:         tasks,
		Runner:        e.Runner,
		CiVersion:     e.Version,
		ElapsedSec:    int64(e.Elapsed / time.Second),
		Phases:        phases,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
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
