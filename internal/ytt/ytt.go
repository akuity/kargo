package ytt

import (
	"os/exec"

	"github.com/pkg/errors"
)

// TODO: Document this
func RenderStrategy(_, baseDir, envDir string) ([]byte, error) {
	cmd := exec.Command("ytt", "--file", baseDir, "--file", envDir)
	yamlBytes, err := cmd.Output()
	return yamlBytes, errors.Wrapf(
		err,
		"error running `%s`",
		cmd.String(),
	)
}
