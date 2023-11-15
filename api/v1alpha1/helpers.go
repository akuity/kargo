package v1alpha1

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func refreshObject(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	nowFunc func() time.Time,
) error {
	patchBytes := []byte(
		fmt.Sprintf(
			`{"metadata":{"annotations":{"%s":"%s"}}}`,
			AnnotationKeyRefresh,
			nowFunc().UTC().Format(time.RFC3339),
		),
	)
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	if err := c.Patch(ctx, obj, patch); err != nil {
		return errors.Wrap(err, "patch annotation")
	}
	return nil
}

func clearRefreshObject(
	ctx context.Context,
	c client.Client,
	obj client.Object,
) error {
	patchBytes := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%s":null}}}`, AnnotationKeyRefresh))
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	if err := c.Patch(ctx, obj, patch); err != nil {
		return errors.Wrap(err, "patch annotation")
	}
	return nil
}
