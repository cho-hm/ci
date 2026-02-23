package parse

import (
	"log"
	"path/filepath"
	"strings"
)

func Run() {
	buildPropPath := buildPropertiesFilePath()
	ctx := TaskContext
	err := resolveValueByTag(buildPropPath, ctx.BuildContexts.Get())
	if err != nil {
		log.Panicf("can not set property from build file : %v", err)
	}

	pubPropPath := publishPropertiesFilePath()
	err = resolveValueByTag(pubPropPath, ctx.PublishContexts.Get())
	if err != nil {
		log.Panicf("can not set property from publish file : %v", err)
	}

	resolveImageNameSuffix()
}

func resolveImageNameSuffix() {
	ctx := TaskContext
	buildContext := ctx.BuildContexts.Get()
	raw := buildContext.RawImageNameSuffix
	each := strings.Split(raw, ":")
	for _, e := range each {
		switch e {
		case "trigger-type":
			buildContext.ImageNameSuffix.TriggerType = true
		case "tag":
			buildContext.ImageNameSuffix.Tag = true
		case "branch":
			buildContext.ImageNameSuffix.Branch = true
		case "sha":
			buildContext.ImageNameSuffix.Sha = true
		case "short-sha":
			buildContext.ImageNameSuffix.ShortSha = true
		case "latest":
			buildContext.ImageNameSuffix.Latest = true
		}
	}
}

func buildPropertiesFilePath() string {
	return filepath.Join(TaskContext.Workspace, TaskContext.EnvDefaults.BuildPropertiesFileSuffix)
}

func publishPropertiesFilePath() string {
	return filepath.Join(TaskContext.Workspace, TaskContext.EnvDefaults.PublishPropertiesFileSuffix)
}
