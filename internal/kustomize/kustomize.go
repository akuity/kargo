package kustomize

import (
	"fmt"
	"os/exec"

	libExec "github.com/akuity/kargo/internal/exec"
)

// SetImage runs `kustomize edit set image ...` in the specified directory.
// The specified directory must already exist and contain a kustomization.yaml
// file.
func SetImage(dir, repo, fqImageRef string) error {
	_, err := libExec.Exec(buildSetImageCmd(dir, repo, fqImageRef))
	return err
}

func buildSetImageCmd(dir, repo, fqImageRef string) *exec.Cmd {
	cmd := exec.Command( // nolint: gosec
		"kustomize",
		"edit",
		"set",
		"image",
		fmt.Sprintf("%s=%s", repo, fqImageRef),
	)
	cmd.Dir = dir
	return cmd
}
