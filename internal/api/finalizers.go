package api

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func EnsureFinalizer(ctx context.Context, c client.Client, obj client.Object) (bool, error) {
	if controllerutil.AddFinalizer(obj, kargoapi.FinalizerName) {
		return true, patchFinalizers(ctx, c, obj)
	}
	return false, nil
}

func RemoveFinalizer(ctx context.Context, c client.Client, obj client.Object) error {
	if controllerutil.RemoveFinalizer(obj, kargoapi.FinalizerName) {
		return patchFinalizers(ctx, c, obj)
	}
	return nil
}

func patchFinalizers(ctx context.Context, c client.Client, obj client.Object) error {
	type objectMeta struct {
		Finalizers []string `json:"finalizers"`
	}
	type patch struct {
		ObjectMeta objectMeta `json:"metadata"`
	}
	data, err := json.Marshal(patch{
		ObjectMeta: objectMeta{
			Finalizers: obj.GetFinalizers(),
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
		return fmt.Errorf("patch finalizers: %w", err)
	}
	return nil
}

func PatchOwnerReferences(ctx context.Context, c client.Client, obj client.Object) error {
	type objectMeta struct {
		OwnerReferences []metav1.OwnerReference `json:"ownerReferences"`
	}
	type patch struct {
		ObjectMeta objectMeta `json:"metadata"`
	}
	data, err := json.Marshal(patch{
		ObjectMeta: objectMeta{
			OwnerReferences: obj.GetOwnerReferences(),
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
		return fmt.Errorf("patch owner references: %w", err)
	}
	return nil
}

func PatchAnnotation(ctx context.Context, c client.Client, obj client.Object, key, value string) error {
	return patchAnnotations(ctx, c, obj, map[string]*string{
		key: ptr.To(value),
	})
}

func deleteAnnotation(ctx context.Context, c client.Client, obj client.Object, key string) error {
	return patchAnnotations(ctx, c, obj, map[string]*string{
		key: nil,
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
