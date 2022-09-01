package helm

import (
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

// TODO: Document this
func RenderStrategy(name, baseDir, envDir string) ([]byte, error) {
	cmd := exec.Command( // nolint: gosec
		"helm",
		"template",
		name,
		baseDir,
		"--values",
		filepath.Join(envDir, "values.yaml"),
	)
	yamlBytes, err := cmd.Output()
	return yamlBytes, errors.Wrapf(
		err,
		"error running `%s`",
		cmd.String(),
	)
}
