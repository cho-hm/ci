package chain

import (
	"ci/core/parse"
	"ci/util/cli"
)

type LogoutBuildChain struct {
	BaseChain
}

func (b LogoutBuildChain) DoChain(context *parse.TaskContexts) error {
	if err := cli.Run("docker", []string{"logout", "ghcr.io"}); err != nil {
		return err
	}
	return b.doNext(context)
}
