package check

import (
	"ci/core/parse"
)

func Run() {
	context := parse.TaskContext
	if context.TaskFlag.Build() {
		go checkBuild()
	}
	if context.TaskFlag.Publish() {
		go checkPublish()
	}
}
