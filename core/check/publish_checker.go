package check

import (
	"ci/core/constant"
	"ci/core/parse"
	"strings"
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

	if ctx.GithubRefType == constant.TAG_VALUE || strings.HasPrefix(ctx.GithubRef, constant.TAG_REF_PREFIX) {
		if !commonCheckTag(ctx, c) {
			return
		}
	}

	if ctx.GithubRefType == constant.BRANCH_VALUE {
		if !commonCheckBranch(ctx, c) {
			return
		}
	}

	pctx.State.Get() <- constant.CONTINUE
}
