package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
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
	kvs := make(map[string]*string, len(keys))
	for _, k := range keys {
		kvs[k] = nil
	}
	return patchAnnotations(ctx, c, obj, kvs)
}

func patchAnnotation(ctx context.Context, c client.Client, obj client.Object, key, value string) error {
	return patchAnnotations(ctx, c, obj, map[string]*string{
		key: ptr.To(value),
	})
}

func patchAnnotations(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	kvs map[string]*string,
) error {
	type objectMeta struct {
		Annotations map[string]*string `json:"annotations"`
	}
	type patch struct {
		ObjectMeta objectMeta `json:"metadata"`
	}
	if len(kvs) == 0 {
		// Do nothing if there are no kv pairs to patch.
		return nil
	}
	data, err := json.Marshal(patch{
		ObjectMeta: objectMeta{
			Annotations: kvs,
		},
	})
	if err != nil {
		return fmt.Errorf("marshal patch data: %w", err)
	}
	if err := c.Patch(
		ctx,
		obj,
		client.RawPatch(types.MergePatchType, data),
	); err != nil {
		return fmt.Errorf("patch annotation: %w", err)
	}
	return nil
}
