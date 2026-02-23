package env

import (
	"ci/core/builds/chain"
	"ci/core/constant"
)

func init() {
	Register("gradle", Gradle{})
}

type Gradle struct{}

func (g Gradle) Defaults() constant.EnvDefaults {
	return constant.EnvDefaults{
		BuildCommand:               "./gradlew clean test bootJar --no-daemon --refresh-dependencies -i",
		PublishCommand:             "./gradlew clean test publish --no-daemon",
		DockerFilePath:             "./.github/Dockerfile",
		ImagePlatform:              "linux/amd64,linux/arm64",
		RawImageNameSuffix:         "trigger-type:tag:branch:sha:short-sha:latest",
		BuildPropertiesFileSuffix:  "ci-ghcr.properties",
		PublishPropertiesFileSuffix: "ci-mvn.properties",
	}
}

func (g Gradle) MkBuildChain() chain.BuildChain {
	logout := chain.LogoutBuildChain{BaseChain: chain.BaseChain{Terminal: true}}
	dockerBuild := chain.RunDockerBuildChain{BaseChain: chain.BaseChain{Next: logout}}
	login := chain.LoginBuildChain{BaseChain: chain.BaseChain{Next: dockerBuild}}
	return chain.BuildBuildChain{BaseChain: chain.BaseChain{Next: login}}
}
