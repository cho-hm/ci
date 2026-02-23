package check

import (
	"ci/core/constant"
	"ci/core/parse"
)

func checkPublish() {
	ctx := parse.TaskContext
	pctx := ctx.PublishContexts.Get()
	if !pctx.IsApplicable {
		pctx.State.Get() <- constant.NOT_APPLICABLE
		return
	}

	c := publishCheckable{pctx}

	ok := commonCheckTriggerType(ctx, c)
	if !ok {
		return
	}

	if ctx.GithubRefType == constant.BRANCH_VALUE {
		if !commonCheckBranch(ctx, c) {
			return
		}
	}

	pctx.State.Get() <- constant.CONTINUE
}
