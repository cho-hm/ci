package env

import (
	"ci/core/builds/chain"
	"ci/core/constant"
)

type Environment interface {
	Defaults() constant.EnvDefaults
	MkBuildChain() chain.BuildChain
}
