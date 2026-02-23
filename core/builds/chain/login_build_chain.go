package chain

import (
	"ci/core/parse"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type LoginBuildChain struct {
	BaseChain
}

func (b LoginBuildChain) DoChain(context *parse.TaskContexts) error {
	if err := context.PatSecret.With(func(token []byte) error {
		cmd := exec.Command("docker", "login", "ghcr.io", "-u", context.GithubActor, "--password-stdin")
		cmd.Stdin = strings.NewReader(string(token))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("docker login failed: %v", err)
		}
		return nil
	}); err != nil {
		return err
	}
	return b.doNext(context)
}
