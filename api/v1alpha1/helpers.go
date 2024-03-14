package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func patchAnnotation(ctx context.Context, c client.Client, obj client.Object, key, value string) error {
	patchBytes := []byte(
		fmt.Sprintf(
			`{"metadata":{"annotations":{"%s":"%s"}}}`,
			key,
			value,
		),
	)
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	if err := c.Patch(ctx, obj, patch); err != nil {
		return fmt.Errorf("patch annotation: %w", err)
	}
	return nil
}

func clearObjectAnnotation(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	annotationKey string,
) error {
	patchBytes := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%s":null}}}`, annotationKey))
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	if err := c.Patch(ctx, obj, patch); err != nil {
		return fmt.Errorf("patch annotation: %w", err)
	}
	return nil
}
