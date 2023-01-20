package kustomize

import (
	"fmt"
	"os/exec"

	libExec "github.com/akuityio/kargo/internal/exec"
)

// SetImage runs `kustomize edit set image ...` in the specified directory.
// The specified directory must already exist and contain a kustomization.yaml
// file.
func SetImage(dir string, repo, tag string) error {
	_, err := libExec.Exec(buildSetImageCmd(dir, repo, tag))
	return err
}

func buildSetImageCmd(dir, repo, tag string) *exec.Cmd {
	cmd := exec.Command( // nolint: gosec
		"kustomize",
		"edit",
		"set",
		"image",
		fmt.Sprintf("%s=%s:%s", repo, repo, tag),
	)
	cmd.Dir = dir
	return cmd
}
