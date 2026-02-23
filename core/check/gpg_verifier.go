package check

import (
	"ci/core/parse"
	"ci/util/git"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func verifyGpgSignature(ctx *parse.TaskContexts, bctx *parse.BuildContexts) error {
	tmpDir, err := os.MkdirTemp("", "gpg-verify-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := cloneGpgRepo(tmpDir, bctx); err != nil {
		return fmt.Errorf("failed to clone gpg repo: %w", err)
	}

	keyFiles, err := collectKeyFiles(tmpDir, bctx)
	if err != nil {
		return fmt.Errorf("failed to collect key files: %w", err)
	}
	if len(keyFiles) == 0 {
		return fmt.Errorf("no GPG key files (.asc, .gpg) found in repo")
	}

	if err := importGpgKeys(keyFiles); err != nil {
		return fmt.Errorf("failed to import gpg keys: %w", err)
	}

	tagName := ctx.GithubRefName
	if _, err := git.Cmd("verify-tag", "-v", "refs/tags/"+tagName); err != nil {
		return fmt.Errorf("gpg signature verification failed for tag %s: %w", tagName, err)
	}

	log.Printf("GPG signature verified for tag: %s\n", tagName)
	return nil
}

func cloneGpgRepo(destDir string, bctx *parse.BuildContexts) error {
	repoUrl := bctx.GpgRepoUrl
	if repoUrl == "" {
		return fmt.Errorf("gpg.repo.url is not set")
	}

	var authUrl string
	bctx.GpgToken.With(func(token []byte) error {
		if len(token) > 0 {
			authUrl = fmt.Sprintf("https://%s@%s", string(token), stripScheme(repoUrl))
		} else {
			authUrl = repoUrl
		}
		return nil
	})

	args := []string{
		"clone", "--depth=1", "--branch", bctx.GpgRepoBranch,
		authUrl, destDir,
	}
	if _, err := git.Cmd(args...); err != nil {
		return err
	}
	return nil
}

func collectKeyFiles(repoDir string, bctx *parse.BuildContexts) ([]string, error) {
	var files []string

	patterns := []string{
		filepath.Join(repoDir, bctx.GpgRepoGpgPath, "*.gpg"),
		filepath.Join(repoDir, bctx.GpgRepoAscPath, "*.asc"),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	return files, nil
}

func importGpgKeys(keyFiles []string) error {
	args := append([]string{"--batch", "--import"}, keyFiles...)
	cmd := exec.Command("gpg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stripScheme(url string) string {
	for _, prefix := range []string{"https://", "http://"} {
		if len(url) > len(prefix) && url[:len(prefix)] == prefix {
			return url[len(prefix):]
		}
	}
	return url
}
