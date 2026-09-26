package notify

import (
	"ci/core/constant"
	"fmt"
	"time"
)

// 플랫폼 공통으로 쓰는 사람이 읽는 요약 표현.

func (e Event) Title() string {
	subject := e.Repository
	if e.Workflow != "" {
		subject = fmt.Sprintf("%s (%s)", e.Repository, e.Workflow)
	}
	return fmt.Sprintf("%s CI %s — %s", e.Outcome.icon(), e.headline(), subject)
}

func (e Event) Lines() []string {
	lines := []string{
		"Ref: " + e.RefName,
		"Commit: " + shortSha(e.Sha),
		"Actor: " + e.Actor,
	}
	if e.Kind == Finished {
		lines = append(lines, "Elapsed: "+e.Elapsed.Round(time.Second).String())
	}
	for _, p := range e.Phases {
		lines = append(lines, p.String())
	}
	if e.Cause != nil {
		lines = append(lines, "Error: "+e.Cause.Error())
	}
	return lines
}

func (e Event) headline() string {
	switch e.Outcome {
	case Succeeded:
		return "succeeded"
	case Failed:
		return "failed"
	default:
		return "started"
	}
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

func shortSha(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
