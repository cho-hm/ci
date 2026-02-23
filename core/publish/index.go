package publish

import (
	"ci/core/constant"
	"ci/core/parse"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Run() constant.PhaseResult {
	ctx := parse.TaskContext
	pctx := ctx.PublishContexts.Get()
	envType := ctx.TaskFlag.EnvType()

	switch envType {
	case "gradle":
		return publishGradle(ctx, pctx)
	case "node":
		return publishNode(ctx, pctx)
	default:
		return constant.PhaseResult{
			Phase:  "publish",
			Status: constant.PhaseFailure,
			Cause:  fmt.Errorf("unsupported env type for publish: %s", envType),
		}
	}
}

func publishGradle(ctx *parse.TaskContexts, pctx *parse.PublishContexts) constant.PhaseResult {
	cmd := pctx.PublishCommand
	if cmd == "" {
		cmd = ctx.EnvDefaults.PublishCommand
	}
	log.Printf("Executing publish command: %s\n", cmd)

	var envVars []string
	envVars = append(envVars, "GPR_USER="+ctx.GithubActor)
	ctx.GithubToken.With(func(token []byte) error {
		envVars = append(envVars, "GPR_TOKEN="+string(token))
		return nil
	})

	if err := shellExec(cmd, ctx.Workspace, envVars...); err != nil {
		return constant.PhaseResult{Phase: "publish", Status: constant.PhaseFailure, Cause: err}
	}
	return constant.PhaseResult{Phase: "publish", Status: constant.PhaseSuccess}
}

func publishNode(ctx *parse.TaskContexts, pctx *parse.PublishContexts) constant.PhaseResult {
	if err := writeNpmrc(ctx); err != nil {
		return constant.PhaseResult{Phase: "publish", Status: constant.PhaseFailure, Cause: err}
	}

	cmd := pctx.PublishCommand
	if cmd == "" {
		cmd = ctx.EnvDefaults.PublishCommand
	}
	log.Printf("Executing publish command: %s\n", cmd)

	if err := shellExec(cmd, ctx.Workspace); err != nil {
		return constant.PhaseResult{Phase: "publish", Status: constant.PhaseFailure, Cause: err}
	}
	return constant.PhaseResult{Phase: "publish", Status: constant.PhaseSuccess}
}

func writeNpmrc(ctx *parse.TaskContexts) error {
	npmrcPath := filepath.Join(ctx.Workspace, ".npmrc")
	var content string
	ctx.GithubToken.With(func(token []byte) error {
		content = fmt.Sprintf("//npm.pkg.github.com/:_authToken=%s\n@%s:registry=https://npm.pkg.github.com\n",
			string(token), strings.Split(ctx.GithubRepository, "/")[0])
		return nil
	})
	return os.WriteFile(npmrcPath, []byte(content), 0600)
}

func shellExec(command string, dir string, extraEnv ...string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
