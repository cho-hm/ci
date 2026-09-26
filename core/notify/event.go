package notify

import (
	"ci/core/constant"
	"strings"
	"time"
)

// Kind - CI 실행 생명주기 중 어느 시점의 알림인지.
type Kind int

const (
	Started Kind = iota
	Finished
)

// Outcome - 알림 시점의 CI 실행 상태.
type Outcome int

const (
	Running Outcome = iota
	Succeeded
	Failed
)

// Event - 플랫폼에 독립적인 알림 모델. 표현(Card)과 플랫폼 형식(Notifier)은 이 모델로부터 만들어진다.
type Event struct {
	Kind          Kind
	Outcome       Outcome
	Workflow      string
	ServerURL     string // 예: https://github.com — 링크 합성에 사용
	Repository    string
	RefName       string
	RefType       string // branch | tag
	EventName     string // push, workflow_dispatch ...
	Sha           string
	CommitMessage string
	Actor         string
	RunURL        string
	RunNumber     string
	RunAttempt    string
	Env           string   // gradle | node
	Tasks         []string // 실행 예정 phase (publish, build)
	Runner        string   // 예: Linux X64
	Version       string   // citool 버전
	Elapsed       time.Duration
	Phases        []constant.PhaseResult
	Cause         error // phase 결과로 표현되지 않는 비정상 종료(panic 등)
}

func (o Outcome) String() string {
	switch o {
	case Succeeded:
		return "success"
	case Failed:
		return "failure"
	default:
		return "running"
	}
}

func (k Kind) String() string {
	if k == Finished {
		return "finished"
	}
	return "started"
}

func (e Event) RepositoryURL() string { return e.link(e.Repository) }

func (e Event) CommitURL() string {
	if e.Sha == "" {
		return ""
	}
	return e.link(e.Repository, "commit", e.Sha)
}

func (e Event) RefURL() string {
	if e.RefName == "" {
		return ""
	}
	return e.link(e.Repository, "tree", e.RefName)
}

func (e Event) ActorURL() string { return e.link(e.Actor) }

func (e Event) ActorAvatarURL() string {
	if url := e.ActorURL(); url != "" {
		return url + ".png?size=64"
	}
	return ""
}

// link - ServerURL 기준 경로를 합성한다. 구성 요소 중 하나라도 비어 있으면 빈 문자열.
func (e Event) link(parts ...string) string {
	if e.ServerURL == "" {
		return ""
	}
	for _, p := range parts {
		if p == "" {
			return ""
		}
	}
	return strings.TrimSuffix(e.ServerURL, "/") + "/" + strings.Join(parts, "/")
}

// FailureCauses - 실패한 phase 의 원인과 비정상 종료 원인.
func (e Event) FailureCauses() []string {
	var causes []string
	for _, p := range e.Phases {
		if p.Failed() && p.Cause != nil {
			causes = append(causes, "["+p.Phase+"] "+p.Cause.Error())
		}
	}
	if e.Cause != nil {
		causes = append(causes, e.Cause.Error())
	}
	return causes
}

// Artifacts - 모든 phase 의 결과물.
func (e Event) Artifacts() []string {
	var artifacts []string
	for _, p := range e.Phases {
		artifacts = append(artifacts, p.Artifacts...)
	}
	return artifacts
}
