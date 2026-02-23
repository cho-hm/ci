package parse

import (
	"ci/core/constant"
	"ci/orch/arg"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type TaskContexts struct {
	Workspace        string
	GithubActor      string
	GithubToken      Secret
	PatSecret        Secret
	GithubRepository string
	GithubRefType    string
	GithubRef        string
	GithubRefName    string
	GithubSha        string
	BuildContexts    *DefaultContextProvider[*BuildContexts]
	PublishContexts  *DefaultContextProvider[*PublishContexts]
	TaskFlag         arg.Flags
	EnvDefaults      constant.EnvDefaults
}

func (t *TaskContexts) init() {
	t.Workspace = getEnv("GITHUB_WORKSPACE")
	t.GithubActor = getEnv("GITHUB_ACTOR")
	t.GithubToken = Secret{getSecret("GITHUB_TOKEN")}
	pat := getSecret("PAT")
	if pat == nil || len(pat) == 0 {
		pat = t.GithubToken.key
	}
	t.PatSecret = Secret{pat}
	t.GithubRepository = getEnv("GITHUB_REPOSITORY")
	t.GithubRefType = getEnv("GITHUB_REF_TYPE")
	t.GithubRef = getEnv("GITHUB_REF")
	t.GithubRefName = getEnv("GITHUB_REF_NAME")
	t.GithubSha = getEnv("GITHUB_SHA")
	t.BuildContexts = &DefaultContextProvider[*BuildContexts]{
		newInstanceFunc: func() *BuildContexts { return new(BuildContexts) },
	}
	t.PublishContexts = &DefaultContextProvider[*PublishContexts]{
		newInstanceFunc: func() *PublishContexts { return new(PublishContexts) },
	}
}

type BuildContexts struct {
	IsApplicable       bool
	GpgToken           Secret
	GpgRepoUrl         string `pkey:"gpg.repo.url"`
	GpgRepoGpgPath     string `pkey:"gpg.repo.gpg.path"`
	GpgRepoAscPath     string `pkey:"gpg.repo.asc.path"`
	GpgRepoBranch      string `pkey:"gpg.repo.branch"`
	DockerFile         string `pkey:"docker.file.path"`
	TriggerType        string `pkey:"trigger.type"`
	TriggerBranch      string `pkey:"trigger.branch"`
	BuildCommand       string `pkey:"build.command"`
	ImagePlatform      string `pkey:"image.platform"`
	RawImageNameSuffix string `pkey:"image.name.suffix"`
	ImageNameSuffix    ImageNameSuffix
	State              *StateChannelProvider
}

func (t *BuildContexts) init() {
	defaults := TaskContext.EnvDefaults
	t.IsApplicable = TaskContext.TaskFlag.Build()
	t.GpgToken = Secret{getSecret("GPG_TOKEN")}
	t.GpgRepoGpgPath = constant.GPG_REPO_GPG_PATH
	t.GpgRepoAscPath = constant.GPG_REPO_ASC_PATH
	t.GpgRepoBranch = constant.GPG_REPO_BRANCH
	t.DockerFile = filepath.Join(TaskContext.Workspace, defaults.DockerFilePath)
	t.TriggerType = constant.TRIGGER_TYPE
	t.TriggerBranch = constant.TRIGGER_BRANCH
	t.BuildCommand = defaults.BuildCommand
	t.ImagePlatform = defaults.ImagePlatform
	t.RawImageNameSuffix = defaults.RawImageNameSuffix
	t.ImageNameSuffix = ImageNameSuffix{}
	t.State = &StateChannelProvider{}
}

type PublishContexts struct {
	IsApplicable   bool
	TriggerType    string `pkey:"trigger.type"`
	TriggerBranch  string `pkey:"trigger.branch"`
	PublishCommand string `pkey:"publish.command"`
	State          *StateChannelProvider
}

func (t *PublishContexts) init() {
	defaults := TaskContext.EnvDefaults
	t.IsApplicable = TaskContext.TaskFlag.Publish()
	t.TriggerType = constant.TRIGGER_TYPE
	t.TriggerBranch = constant.TRIGGER_BRANCH
	t.PublishCommand = defaults.PublishCommand
	t.State = &StateChannelProvider{}
}

type Secret struct {
	key []byte
}

func (s Secret) String() string               { return constant.REDACTION }
func (s Secret) GoString() string             { return constant.REDACTION }
func (s Secret) MarshalJSON() ([]byte, error) { return json.Marshal(constant.REDACTION) }
func (s Secret) MarshalText() ([]byte, error) { return []byte(constant.REDACTION), nil }
func (s *Secret) With(fn func([]byte) error) error {
	return fn(s.key)
}

type ImageNameSuffix struct {
	TriggerType bool
	Tag         bool
	Branch      bool
	Sha         bool
	ShortSha    bool
	Latest      bool
}

func (suffix ImageNameSuffix) ToSlice(refName string, triggerType string, sha string) []string {
	ret := make([]string, 0, 6)
	if suffix.TriggerType {
		ret = append(ret, triggerType)
	}
	if suffix.Tag && strings.Contains(strings.ToLower(triggerType), constant.TAG_VALUE) {
		ret = append(ret, refName)
	}
	if suffix.Branch {
		ret = append(ret, refName)
	}
	if suffix.Sha || suffix.ShortSha {
		if suffix.Sha {
			ret = append(ret, sha)
		}
		if suffix.ShortSha {
			ret = append(ret, string([]rune(sha)[:7]))
		}
	}
	if suffix.Latest {
		ret = append(ret, "latest")
	}
	return ret
}

func getSecret(key string) []byte {
	return []byte(getEnv(key))
}
func getEnv(key string) string {
	return os.Getenv(key)
}
