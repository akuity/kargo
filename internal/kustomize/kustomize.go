package kustomize

import (
	"fmt"
	"os/exec"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func SetImage(dir string, image api.Image, logger *log.Entry) error {
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
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "error setting image")
	}
	logger.Debug("ran kustomize edit set image")
	return nil
}

func Build(branch, dir string, logger *log.Entry) ([]byte, error) {
	cmd := exec.Command("kustomize", "build")
	cmd.Dir = dir
	yamlBytes, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error rendering YAML for branch %q",
			branch,
		)
	}
	logger.Debug("ran kustomize build")
	return yamlBytes, nil
}
