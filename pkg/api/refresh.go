package api

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// RefreshObject forces reconciliation of a Kubernetes object by setting an
// annotation on the object, causing the controller to reconcile it. Currently,
// the annotation value is the timestamp of the request, but in the future
// may include additional metadata/context necessary for the request.
func RefreshObject(ctx context.Context, c client.Client, obj client.Object) error {
	return patchAnnotation(
		ctx,
		c,
		obj,
		kargoapi.AnnotationKeyRefresh,
		time.Now().Format(time.RFC3339),
	)
}
