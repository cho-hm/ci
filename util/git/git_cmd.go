package git

import (
	"bytes"
	"fmt"
	"os/exec"
)

func Cmd(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %v failed: %w: %s\n",
			args, err, stderr.String())
	}
	return out.String(), nil
}
