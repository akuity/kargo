package v1alpha1

import (
	"context"
	"fmt"
	"time"

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
		return nil, fmt.Errorf(
			"error getting Stage %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
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
	stage := &Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespacedName.Namespace,
			Name:      namespacedName.Name,
		},
	}
	if err := refreshObject(ctx, c, stage, time.Now); err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}
	return stage, nil
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
	newStage := Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stage.Name,
			Namespace: stage.Namespace,
		},
	}
	return clearRefreshObject(ctx, c, &newStage)
}
