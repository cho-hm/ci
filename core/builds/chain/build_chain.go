package chain

import (
	"ci/core/parse"
	"os"
	"os/exec"
)

type BuildChain interface {
	DoChain(context *parse.TaskContexts) error
}

type BaseChain struct {
	Next     BuildChain
	Terminal bool
}

func (b BaseChain) doNext(context *parse.TaskContexts) error {
	if b.Terminal {
		return nil
	}
	return b.Next.DoChain(context)
}

func mapStd(cmd *exec.Cmd) {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
}
