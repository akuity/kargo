package kustomize

import (
	"fmt"
	"os/exec"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/pkg/errors"
)

// TODO: Document this
type RenderStrategy struct{}

// SetImage runs `kustomize edit set image ...` in the specified directory.
func (r *RenderStrategy) SetImage(dir string, image api.Image) error {
	cmd := exec.Command( // nolint: gosec
		"kustomize",
		"edit",
		"set",
		"image",
		fmt.Sprintf(
			"%s=%s:%s",
			image.Repo,
			image.Repo,
			image.Tag,
		),
	)
	cmd.Dir = dir
	return errors.Wrapf(
		cmd.Run(),
		"error running kustomize set image in directory %q",
		dir,
	)
}

// Build runs `kustomize build` in the specified directory and returns an array
// of bytes containing the fully rendered YAML.
func (r *RenderStrategy) Build(dir string) ([]byte, error) {
	cmd := exec.Command("kustomize", "build")
	cmd.Dir = dir
	yamlBytes, err := cmd.Output()
	return yamlBytes, errors.Wrapf(
		err,
		"error running kustomize build in directory %q",
		dir,
	)
}
