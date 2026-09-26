package notify

import "strings"

// texts - 알림 문구 카탈로그. 새 언어는 catalog 에 항목을 추가한다.
type texts struct {
	Started, Succeeded, Failed      string
	PhaseSucceeded, PhaseFailed     string
	PhaseSkipped                    string
	Attempt                         string // "%s" 자리에 재시도 횟수
	Repository, Ref, Trigger, Env   string
	Tasks, Duration, Commit, Phases string
	Artifacts, Error                string
	ViewRun, ViewCommit             string
	By                              string // "%s" 자리에 실행자
}

const defaultLang = "en"

var catalog = map[string]texts{
	"en": {
		Started: "started", Succeeded: "succeeded", Failed: "failed",
		PhaseSucceeded: "succeeded", PhaseFailed: "failed", PhaseSkipped: "skipped",
		Attempt:    "attempt %s",
		Repository: "Repository", Ref: "Ref", Trigger: "Trigger", Env: "Env",
		Tasks: "Tasks", Duration: "Duration", Commit: "Commit", Phases: "Phases",
		Artifacts: "Images", Error: "Error",
		ViewRun: "View run", ViewCommit: "View commit",
		By: "by %s",
	},
	"ko": {
		Started: "시작", Succeeded: "성공", Failed: "실패",
		PhaseSucceeded: "성공", PhaseFailed: "실패", PhaseSkipped: "건너뜀",
		Attempt:    "재시도 %s",
		Repository: "저장소", Ref: "Ref", Trigger: "트리거", Env: "환경",
		Tasks: "작업", Duration: "소요 시간", Commit: "커밋", Phases: "단계",
		Artifacts: "이미지", Error: "오류",
		ViewRun: "실행 보기", ViewCommit: "커밋 보기",
		By: "%s 님이 실행",
	},
}

// textsOf - 지원하지 않는 언어면 false 와 함께 기본 언어 문구를 반환한다.
func textsOf(lang string) (texts, bool) {
	if lang == "" {
		return catalog[defaultLang], true
	}
	t, ok := catalog[strings.ToLower(strings.TrimSpace(lang))]
	if !ok {
		return catalog[defaultLang], false
	}
	return t, true
}
