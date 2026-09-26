package constant

import (
	"fmt"
	"time"
)

type PhaseResult struct {
	Phase     string
	Status    PhaseStatus
	Cause     error
	Reason    string
	Elapsed   time.Duration // phase 실행 소요 시간 (스킵 시 0)
	Artifacts []string      // phase 가 만들어 낸 결과물 (예: 푸시된 이미지 참조)
}

type PhaseStatus int

const (
	PhaseSuccess PhaseStatus = iota
	PhaseFailure
	PhaseSkipped
)

func (r PhaseResult) Failed() bool {
	return r.Status == PhaseFailure
}

func (r PhaseResult) String() string {
	switch r.Status {
	case PhaseSuccess:
		return fmt.Sprintf("[%s] succeeded", r.Phase)
	case PhaseFailure:
		return fmt.Sprintf("[%s] FAILED: %v", r.Phase, r.Cause)
	case PhaseSkipped:
		if r.Reason != "" {
			return fmt.Sprintf("[%s] SKIPPED — %s", r.Phase, r.Reason)
		}
		return fmt.Sprintf("[%s] SKIPPED", r.Phase)
	}
	return ""
}
