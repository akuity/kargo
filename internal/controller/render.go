package controller

import api "github.com/akuityio/k8sta/api/v1alpha1"

// TODO: Document this
type RenderStrategy interface {
	// TODO: Document this
	SetImage(dir string, image api.Image) error
	// TODO: Document this
	Build(dir string) ([]byte, error)
}
