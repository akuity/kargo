package ytt

import (
	"fmt"
	"os/exec"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/pkg/errors"
)

// TODO: Document this
type RenderStrategy struct{}

// SetImage runs `yq eval --inplace ...` in the specified directory.
func (r *RenderStrategy) SetImage(dir string, image api.Image) error {
	cmd := exec.Command( // nolint: gosec
		"yq",
		"eval",
		"--inplace",
		fmt.Sprintf(
			`.images["%s"]="%s:%s"`,
			image.Repo,
			image.Repo,
			image.Tag,
		),
		"values.yaml",
	)
	cmd.Dir = dir
	return errors.Wrapf(
		cmd.Run(),
		"error running `yq eval ...` in directory %q",
		dir,
	)
}

// Build runs `ytt` to combine templated base configuration from baseDir with
// values from the overlay in envDir and returns an array of bytes containing
// the fully rendered YAML.
func (r *RenderStrategy) Build(baseDir, envDir string) ([]byte, error) {
	cmd := exec.Command("ytt", "--file", baseDir, "--file", envDir)
	yamlBytes, err := cmd.Output()
	return yamlBytes, errors.Wrapf(
		err,
		"error running `%s`",
		cmd.String(),
	)
}
