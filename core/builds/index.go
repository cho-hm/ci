package build

import (
	"ci/core/constant"
	"ci/core/env"
	"ci/core/parse"
)

func Run() constant.PhaseResult {
	e := env.Of(parse.TaskContext.TaskFlag.EnvType())
	if err := e.MkBuildChain().DoChain(parse.TaskContext); err != nil {
		return constant.PhaseResult{Phase: "build", Status: constant.PhaseFailure, Cause: err}
	}
	return constant.PhaseResult{Phase: "build", Status: constant.PhaseSuccess}
}
