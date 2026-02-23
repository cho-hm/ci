package check

import (
	"ci/core/constant"
	"ci/core/parse"
	"log"
	"strings"
)

type checkable interface {
	triggerType() string
	triggerBranch() string
	stateCh() chan int
}

type buildCheckable struct{ ctx *parse.BuildContexts }

func (b buildCheckable) triggerType() string   { return b.ctx.TriggerType }
func (b buildCheckable) triggerBranch() string { return b.ctx.TriggerBranch }
func (b buildCheckable) stateCh() chan int     { return b.ctx.State.Get() }

type publishCheckable struct{ ctx *parse.PublishContexts }

func (p publishCheckable) triggerType() string   { return p.ctx.TriggerType }
func (p publishCheckable) triggerBranch() string { return p.ctx.TriggerBranch }
func (p publishCheckable) stateCh() chan int     { return p.ctx.State.Get() }

func commonCheckTriggerType(ctx *parse.TaskContexts, c checkable) bool {
	tt := c.triggerType()
	if tt == constant.SIGNED_TAG_VALUE || tt == constant.TAG_VALUE {
		if ctx.GithubRefType != constant.TAG_VALUE && !strings.HasPrefix(ctx.GithubRef, constant.TAG_REF_PREFIX) {
			logExpectFail(tt, ctx.GithubRefType, "trigger type")
			c.stateCh() <- constant.SILENTLY
			return false
		}
	} else if ctx.GithubRefType == constant.TAG_VALUE || strings.HasPrefix(ctx.GithubRef, constant.TAG_REF_PREFIX) {
		logExpectFail(tt, ctx.GithubRefType, "trigger type")
		c.stateCh() <- constant.SILENTLY
		return false
	}
	return true
}

func commonCheckBranch(ctx *parse.TaskContexts, c checkable) bool {
	branchName := strings.TrimSpace(ctx.GithubRefName)
	if len(branchName) == 0 {
		logExpectFail(c.triggerBranch(), branchName, "trigger branch")
		c.stateCh() <- constant.SILENTLY
		return false
	}
	for _, allowed := range strings.Split(c.triggerBranch(), ":") {
		if strings.TrimSpace(allowed) == branchName {
			return true
		}
	}
	logExpectFail(c.triggerBranch(), branchName, "trigger branch")
	c.stateCh() <- constant.SILENTLY
	return false
}

func logExpectFail(expect string, actual string, checkType string) {
	log.Printf("Expected `%s`=%s, but actual: %s\n", checkType, expect, actual)
	if len(checkType) > 0 {
		log.Printf("Invalid %s, reject ci silently...\n", checkType)
	}
}
