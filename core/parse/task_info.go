package parse

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ImageRefs - 이번 실행에서 빌드/푸시할 GHCR 이미지 참조 목록.
func (t *TaskContexts) ImageRefs() []string {
	buildCtx := t.BuildContexts.Get()
	base := fmt.Sprintf("ghcr.io/%s", strings.ToLower(t.GithubRepository))
	suffixes := buildCtx.ImageNameSuffix.ToSlice(t.GithubRefName, buildCtx.TriggerType, t.GithubSha)
	refs := make([]string, 0, len(suffixes))
	for _, s := range suffixes {
		refs = append(refs, base+":"+s)
	}
	return refs
}

// HeadCommitMessage - 워크플로우 이벤트 페이로드의 head commit 메시지 첫 줄. 없으면 빈 문자열.
func (t *TaskContexts) HeadCommitMessage() string {
	if t.GithubEventPath == "" {
		return ""
	}
	raw, err := os.ReadFile(t.GithubEventPath)
	if err != nil {
		return ""
	}
	var event struct {
		HeadCommit struct {
			Message string `json:"message"`
		} `json:"head_commit"`
	}
	if json.Unmarshal(raw, &event) != nil {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(event.HeadCommit.Message, "\n", 2)[0])
}
