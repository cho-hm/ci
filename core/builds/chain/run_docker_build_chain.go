package chain

import (
	"ci/core/parse"
	"ci/util/cli"
	"fmt"
	"strings"
)

type RunDockerBuildChain struct {
	BaseChain
}

func (b RunDockerBuildChain) DoChain(context *parse.TaskContexts) error {
	buildCtx := context.BuildContexts.Get()
	dockerfilePath := buildCtx.DockerFile
	baseImage := strings.ToLower(context.GithubRepository)
	ghcrBase := fmt.Sprintf("ghcr.io/%s", baseImage)

	args := []string{
		"buildx", "build",
		"-f", dockerfilePath,
		"--platform", buildCtx.ImagePlatform,
		"--provenance=false",
		"--push",
	}

	for _, s := range buildCtx.ImageNameSuffix.ToSlice(context.GithubRefName, buildCtx.TriggerType, context.GithubSha) {
		args = append(args, "-t", fmt.Sprintf("%s:%s", ghcrBase, s))
	}

	args = append(args, context.Workspace)

	if err := cli.Run("docker", args); err != nil {
		return err
	}
	return b.doNext(context)
}
