package kustomize

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/common/file"
	"github.com/pkg/errors"
)

var kustomizationBytes = []byte(
	`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- all.yaml`,
)

// TODO: Document this
func EnsurePrerenderDir(dir string) error {
	// Ensure the existence of the directory
	if err := file.EnsureDirectory(dir); err != nil {
		return errors.Wrapf(err, "error creating directory %q", dir)
	}
	kustomizationFile := filepath.Join(dir, "kustomization.yaml")
	// Ensure the existence of kustomization.yaml
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
func RenderStrategy(_, _, envDir string) ([]byte, error) {
	cmd := exec.Command("kustomize", "build")
	cmd.Dir = envDir
	yamlBytes, err := cmd.Output()
	return yamlBytes, errors.Wrapf(
		err,
		"error running `%s` in directory %q",
		cmd.String(),
		cmd.Dir,
	)
}
