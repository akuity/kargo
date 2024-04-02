package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func AddFinalizer(ctx context.Context, c client.Client, obj client.Object) error {
	if controllerutil.AddFinalizer(obj, FinalizerName) {
		patchBytes := []byte(`{"metadata":{"finalizers":[`)
		for i, finalizer := range obj.GetFinalizers() {
			if i > 0 {
				patchBytes = append(patchBytes, ',')
			}
			patchBytes = append(patchBytes, fmt.Sprintf("%q", finalizer)...)
		}
		patchBytes = append(patchBytes, "]}}"...)
		if err := c.Patch(
			ctx,
			obj,
			client.RawPatch(types.MergePatchType, patchBytes),
		); err != nil {
			return fmt.Errorf("patch annotation: %w", err)
		}
		return nil
	}
	return nil
}

func ClearAnnotations(ctx context.Context, c client.Client, obj client.Object, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	patchBytes := []byte(`{"metadata":{"annotations":{`)
	for i, key := range keys {
		if i > 0 {
			patchBytes = append(patchBytes, ',')
		}
		patchBytes = append(patchBytes, fmt.Sprintf(`"%s":null`, key)...)
	}
	patchBytes = append(patchBytes, "}}}"...)
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	if err := c.Patch(ctx, obj, patch); err != nil {
		return fmt.Errorf("patch annotation: %w", err)
	}
	return nil
}

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

func AddV05CompatibilityLabel(
	ctx context.Context,
	c client.Client,
	obj client.Object,
) error {
	patchBytes := []byte(
		fmt.Sprintf(
			`{"metadata":{"labels":{"%s":"true"}}}`,
			V05CompatibilityLabelKey,
		),
	)
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	return c.Patch(ctx, obj, patch)
}
