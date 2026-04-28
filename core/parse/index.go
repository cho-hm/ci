package parse

import (
	"errors"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

func Run() {
	ctx := TaskContext

	if buildCtx := ctx.BuildContexts.Get(); buildCtx.IsApplicable {
		loadProperties("build", buildPropertiesFilePath(), buildCtx)
	}

	if publishCtx := ctx.PublishContexts.Get(); publishCtx.IsApplicable {
		loadProperties("publish", publishPropertiesFilePath(), publishCtx)
	}

	resolveImageNameSuffix()
}

func loadProperties(label, path string, target any) {
	err := resolveValueByTag(path, target)
	if errors.Is(err, fs.ErrNotExist) {
		log.Printf("%s properties file not found, using defaults: %s", label, path)
		return
	}
	if err != nil {
		log.Panicf("can not set property from %s file : %v", label, err)
	}
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
