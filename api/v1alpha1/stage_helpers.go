package v1alpha1

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetStage returns a pointer to the Stage resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetStage(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Stage, error) {
	stage := Stage{}
	if err := c.Get(ctx, namespacedName, &stage); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, errors.Wrapf(
			err,
			"error getting Stage %q in namespace %q",
			namespacedName.Name,
			namespacedName.Namespace,
		)
	}
	return &stage, nil
}

// RefreshStage forces reconciliation of a Stage by setting an annotation
// on the Stage, causing the controller to reconcile it. Currently, the
// annotation value is the timestamp of the request, but might in the
// future include additional metadata/context necessary for the request.
func RefreshStage(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Stage, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	stage := Stage{}
	stage.Name = namespacedName.Name
	stage.Namespace = namespacedName.Namespace
	patchBytes := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s"}}}`, AnnotationKeyRefresh, now))
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	err := c.Patch(ctx, &stage, patch)
	if err != nil {
		return nil, err
	}
	return &stage, nil
}

// ClearStageRefresh is called by the Stage controller to clear the refresh
// annotation on the Stage (if present). A client (e.g. UI) who requested a
// Stage refresh, can wait until the annotation is cleared, to understand that
// the controller successfully reconciled the Stage after the refresh request.
func ClearStageRefresh(
	ctx context.Context,
	c client.Client,
	stage *Stage,
) error {
	if stage.Annotations == nil {
		return nil
	}
	if _, ok := stage.Annotations[AnnotationKeyRefresh]; !ok {
		return nil
	}
	patchBytes := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%s":null}}}`, AnnotationKeyRefresh))
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	newStage := Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stage.Name,
			Namespace: stage.Namespace,
		},
	}
	return c.Patch(ctx, &newStage, patch)
}
