package check

import (
	"ci/core/constant"
	"ci/core/parse"
	"ci/util/git"
	"log"
	"strings"
)

func checkBuild() {
	ctx := parse.TaskContext
	bctx := ctx.BuildContexts.Get()
	if !bctx.IsApplicable {
		bctx.State.Get() <- constant.NOT_APPLICABLE
		return
	}

	c := buildCheckable{bctx}

	ok := commonCheckTriggerType(ctx, c)
	if !ok {
		return
	}

	if bctx.TriggerType == constant.SIGNED_TAG_VALUE {
		if !checkSignedTag(ctx, bctx) {
			return
		}
	}

	if ctx.GithubRefType == constant.BRANCH_VALUE {
		if !commonCheckBranch(ctx, c) {
			return
		}
	}

	bctx.State.Get() <- constant.CONTINUE
}

func checkSignedTag(ctx *parse.TaskContexts, bctx *parse.BuildContexts) bool {
	// 원격 태그 동기화
	if _, err := git.Cmd("fetch", "--force", "--prune", "origin", "+refs/tags/*:refs/tags/*"); err != nil {
		log.Printf("Failed to fetch remote tags: %v\n", err)
		bctx.State.Get() <- constant.ERROR
		return false
	}

	sha := ctx.GithubSha

	// 현재 커밋에 태그가 존재하는지 확인
	if _, err := git.Cmd("tag", "--points-at", sha); err != nil {
		log.Printf("No tag found pointing at SHA %s\n", sha)
		bctx.State.Get() <- constant.SILENTLY
		return false
	}

	// 태그 이름 exact-match 확인
	desc, err := git.Cmd("describe", "--exact-match", "--tags", sha)
	if err != nil || strings.TrimSpace(desc) != ctx.GithubRefName {
		logExpectFail(ctx.GithubRefName, strings.TrimSpace(desc), "tag exact match")
		bctx.State.Get() <- constant.SILENTLY
		return false
	}

	// annotated tag 확인
	tagPath := "refs/tags/" + ctx.GithubRefName
	typ, err := git.Cmd("cat-file", "-t", tagPath)
	if err != nil || strings.TrimSpace(typ) != "tag" {
		logExpectFail(bctx.TriggerType, "not annotated tag", "signed tag")
		bctx.State.Get() <- constant.SILENTLY
		return false
	}

	// GPG 서명 존재 확인
	body, err := git.Cmd("cat-file", "-p", tagPath)
	if err != nil || !strings.Contains(body, "BEGIN PGP SIGNATURE") {
		logExpectFail(bctx.TriggerType, "unsigned tag", "signed tag")
		bctx.State.Get() <- constant.SILENTLY
		return false
	}

	// GPG 키 import + 서명 검증
	if err := verifyGpgSignature(ctx, bctx); err != nil {
		log.Printf("GPG verification failed: %v\n", err)
		bctx.State.Get() <- constant.ERROR
		return false
	}

	return true
}
