package chain

import (
	"ci/core/parse"
	"ci/util/cli"
)

type RunDockerBuildChain struct {
	BaseChain
}

func (b RunDockerBuildChain) DoChain(context *parse.TaskContexts) error {
	buildCtx := context.BuildContexts.Get()
	dockerfilePath := buildCtx.DockerFile

	args := []string{
		"buildx", "build",
		"-f", dockerfilePath,
		"--platform", buildCtx.ImagePlatform,
		"--provenance=false",
		"--push",
	}

	for _, ref := range context.ImageRefs() {
		args = append(args, "-t", ref)
	}

	args = append(args, context.Workspace)

	if err := cli.Run("docker", args); err != nil {
		return err
	}
	return b.doNext(context)
}
