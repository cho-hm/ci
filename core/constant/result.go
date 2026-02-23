package constant

import "fmt"

type PhaseResult struct {
	Phase  string
	Status PhaseStatus
	Cause  error
	Reason string
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
