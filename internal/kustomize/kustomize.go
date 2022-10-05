package kustomize

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/common/file"
)

var kustomizationBytes = []byte(
	`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ephemeral.yaml
`,
)

// EnsureBookkeeperDir ensures the existence of a .bookkeeper directory and the
// kustomize configuration required to perform last-mile rendering.
func EnsureBookkeeperDir(dir string) error {
	// Ensure the existence of the directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrapf(err, "error ensuring existence of directory %q", dir)
	}

	// Ensure the existence of kustomization.yaml
	kustomizationFile := filepath.Join(dir, "kustomization.yaml")
	if exists, err := file.Exists(kustomizationFile); err != nil {
		return errors.Wrapf(
			err,
			"error checking for existence of %q",
			kustomizationFile,
		)
	} else if !exists {
		if err = os.WriteFile( // nolint: gosec
			kustomizationFile,
			kustomizationBytes,
			0644,
		); err != nil {
			return errors.Wrapf(
				err,
				"error writing to %q",
				kustomizationFile,
			)
		}
	}

	return nil
}

// SetImage runs `kustomize edit set image ...` in the specified directory.
// The specified directory must already exist and contain a kustomization.yaml
// file.
func SetImage(dir string, image api.Image) error {
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

// TODO: Document this
func Render(dir string) ([]byte, error) {
	cmd := exec.Command("kustomize", "build")
	cmd.Dir = dir
	yamlBytes, err := cmd.Output()
	return yamlBytes, errors.Wrapf(
		err,
		"error running `%s` in directory %q",
		cmd.String(),
		cmd.Dir,
	)
}
