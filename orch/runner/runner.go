package runner

import (
	build "ci/core/builds"
	"ci/core/check"
	"ci/core/constant"
	"ci/core/env"
	"ci/core/parse"
	"ci/core/publish"
	"ci/orch/arg"
	"log"
	"os"
	"time"
)

// Run - orch.runner.Run()
// Orchestration tasks.
func Run() {
	log.Printf("=== Welcome ===\n\nStart Integration!")
	var results []constant.PhaseResult
	notification := newNotification()
	defer notification.finishedOnPanic(&results)

	var flag, tasks = arg.Flag(), arg.Task()
	parse.TaskContext.EnvDefaults = env.Of(flag.EnvType()).Defaults()
	parse.TaskContext.TaskFlag = flag
	notification.started()
	step := 1
	if flag.Parse() {
		printStartLog(&step, tasks, "parse properties")
		parse.Run()
		printEndLog()
	}

	context := parse.TaskContext

	if flag.Check() {
		printStartLog(&step, tasks, "check commit")
		check.Run()
	}

	if flag.Publish() {
		pctx := context.PublishContexts.Get()
		ch := pctx.State.Get()
		printStartLog(&step, tasks, "publish")
		if ch == nil {
			panic("Need check process.")
		}
		switch state := <-ch; state {
		case constant.CONTINUE:
			results = append(results, timed(publish.Run))
		default:
			results = append(results, constant.PhaseResult{Phase: "publish", Status: constant.PhaseSkipped, Reason: skipReason(state)})
		}
		printEndLog()
	}

	if flag.Build() {
		bctx := context.BuildContexts.Get()
		ch := bctx.State.Get()
		printStartLog(&step, tasks, "build and deploy")
		if ch == nil {
			panic("Need check process.")
		}
		switch state := <-ch; state {
		case constant.CONTINUE:
			results = append(results, timed(build.Run))
		default:
			results = append(results, constant.PhaseResult{Phase: "build", Status: constant.PhaseSkipped, Reason: skipReason(state)})
		}
		printEndLog()
	}

	reportResults(results)
	notification.finished(results)
	if hasFailed(results) {
		os.Exit(1)
	}
	log.Printf("%c: %d Tasks all done!", '\u2714', tasks)
}

// timed - phase 실행 소요 시간을 결과에 기록한다.
func timed(run func() constant.PhaseResult) constant.PhaseResult {
	startedAt := time.Now()
	result := run()
	result.Elapsed = time.Since(startedAt)
	return result
}

func printStartLog(step *int, tasks int, message string) {
	log.Printf("Start step %d/%d: %s...\n", *step, tasks, message)
	*step = *step + 1
}

func printEndLog() {
	log.Printf("done!\n")
}

func hasFailed(results []constant.PhaseResult) bool {
	for _, r := range results {
		if r.Failed() {
			return true
		}
	}
	return false
}

func skipReason(state int) string {
	switch state {
	case constant.NOT_APPLICABLE:
		return "not applicable"
	case constant.SILENTLY:
		return "check not passed"
	case constant.ERROR:
		return "check error"
	default:
		return "unknown"
	}
}

func reportResults(results []constant.PhaseResult) {
	if len(results) == 0 {
		return
	}
	log.Println("=== Results ===")
	for _, r := range results {
		log.Println(r.String())
	}
}
