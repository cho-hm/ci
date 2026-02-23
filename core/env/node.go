package env

import (
	"ci/core/builds/chain"
	"ci/core/constant"
)

func init() {
	Register("node", Node{})
}

type Node struct{}

func (n Node) Defaults() constant.EnvDefaults {
	return constant.EnvDefaults{
		BuildCommand:               "npm ci && npm run build",
		PublishCommand:             "npm publish --registry=https://npm.pkg.github.com",
		DockerFilePath:             "./.github/Dockerfile",
		ImagePlatform:              "linux/amd64,linux/arm64",
		RawImageNameSuffix:         "trigger-type:tag:branch:short-sha:latest",
		BuildPropertiesFileSuffix:  "ci-ghcr.properties",
		PublishPropertiesFileSuffix: "ci-npm.properties",
	}
}

func (n Node) MkBuildChain() chain.BuildChain {
	logout := chain.LogoutBuildChain{BaseChain: chain.BaseChain{Terminal: true}}
	dockerBuild := chain.RunDockerBuildChain{BaseChain: chain.BaseChain{Next: logout}}
	login := chain.LoginBuildChain{BaseChain: chain.BaseChain{Next: dockerBuild}}
	return chain.BuildBuildChain{BaseChain: chain.BaseChain{Next: login}}
}
