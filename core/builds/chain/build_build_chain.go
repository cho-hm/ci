package chain

import (
	"ci/core/parse"
	"os/exec"
)

type BuildBuildChain struct {
	BaseChain
}

func (b BuildBuildChain) DoChain(context *parse.TaskContexts) error {
	command := context.BuildContexts.Get().BuildCommand
	cmd := exec.Command("bash", "-lc", command)
	mapStd(cmd)
	if err := cmd.Run(); err != nil {
		return err
	}
	return b.doNext(context)
}
