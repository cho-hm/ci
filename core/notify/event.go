package notify

import (
	"ci/core/constant"
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

// Event - 플랫폼에 독립적인 알림 모델. 각 Notifier 가 자기 플랫폼 형식으로 변환한다.
type Event struct {
	Kind       Kind
	Outcome    Outcome
	Workflow   string
	Repository string
	RefName    string
	Sha        string
	Actor      string
	RunURL     string
	Elapsed    time.Duration
	Phases     []constant.PhaseResult
	Cause      error // phase 결과로 표현되지 않는 비정상 종료(panic 등)
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
