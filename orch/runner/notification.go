package runner

import (
	"ci/core/constant"
	"ci/core/notify"
	"ci/core/parse"
	"fmt"
	"log"
	"runtime/debug"
	"time"
)

// notification - 실행 컨텍스트를 notify.Event 로 옮겨 시작/종료 알림을 보낸다.
type notification struct {
	sender    notify.Sender
	startedAt time.Time
}

func newNotification() *notification {
	ctx := parse.TaskContext
	var webhookUrl string
	ctx.WebhookUrl.With(func(key []byte) error {
		webhookUrl = string(key)
		return nil
	})
	return &notification{
		sender:    notify.New(notify.Config{WebhookURL: webhookUrl, WebhookType: ctx.WebhookType}),
		startedAt: time.Now(),
	}
}

func (n *notification) started() {
	n.sender.Send(n.event(notify.Started, notify.Running, nil, nil))
}

func (n *notification) finished(results []constant.PhaseResult) {
	outcome := notify.Succeeded
	if hasFailed(results) {
		outcome = notify.Failed
	}
	n.sender.Send(n.event(notify.Finished, outcome, results, nil))
}

// finishedOnPanic - defer 로 호출. panic 으로 비정상 종료될 때 실패 알림을 보낸 뒤 panic 을 이어서 전파한다.
func (n *notification) finishedOnPanic(results *[]constant.PhaseResult) {
	r := recover()
	if r == nil {
		return
	}
	log.Printf("panic: %v\n%s", r, debug.Stack())
	n.sender.Send(n.event(notify.Finished, notify.Failed, *results, fmt.Errorf("panic: %v", r)))
	panic(r)
}

func (n *notification) event(kind notify.Kind, outcome notify.Outcome, results []constant.PhaseResult, cause error) notify.Event {
	ctx := parse.TaskContext
	return notify.Event{
		Kind:       kind,
		Outcome:    outcome,
		Workflow:   ctx.GithubWorkflow,
		Repository: ctx.GithubRepository,
		RefName:    ctx.GithubRefName,
		Sha:        ctx.GithubSha,
		Actor:      ctx.GithubActor,
		RunURL:     runUrl(ctx),
		Elapsed:    time.Since(n.startedAt),
		Phases:     results,
		Cause:      cause,
	}
}

func runUrl(ctx *parse.TaskContexts) string {
	if ctx.GithubServerUrl == "" || ctx.GithubRepository == "" || ctx.GithubRunId == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/actions/runs/%s", ctx.GithubServerUrl, ctx.GithubRepository, ctx.GithubRunId)
}
