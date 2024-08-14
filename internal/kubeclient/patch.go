package kubeclient

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HasStatus[S any] interface {
	client.Object
	GetStatus() S
}

// PatchStatus patches evaluate changes applied by the callback to the status of a resource
// and patches resource status if there are any changes.
func PatchStatus[T HasStatus[S], S any](
	ctx context.Context, kubeClient client.Client, resource T, update func(status S)) error {

	originalJSON, err := json.Marshal(resource.GetStatus())
	if err != nil {
		return err
	}

	var updated S
	if err = json.Unmarshal(originalJSON, &updated); err != nil {
		return err
	}
	update(updated)

	modifiedJSON, err := json.Marshal(updated)
	if err != nil {
		return err
	}

	statusPatch, err := jsonpatch.CreateMergePatch(originalJSON, modifiedJSON)
	if err != nil {
		return err
	}

	patchMap := map[string]any{}
	if err = json.Unmarshal(statusPatch, &patchMap); err != nil {
		return err
	}
	if len(patchMap) == 0 {
		return nil
	}

	patch, err := json.Marshal(map[string]any{
		"status": patchMap,
	})
	if err != nil {
		return err
	}
	return kubeClient.Status().Patch(ctx, resource, client.RawPatch(types.MergePatchType, patch))
}

type ObjectWithKind interface {
	client.Object
	schema.ObjectKind
}

// UnstructuredPatchFn is a function which modifies the destination
// unstructured object based on the source unstructured object.
type UnstructuredPatchFn func(src, dest unstructured.Unstructured) error

// PatchUnstructured patches a Kubernetes object using unstructured objects.
// It fetches the object from the API server, applies modifications via the
// provided UnstructuredPatchFn, and patches the object back to the server.
//
// The UnstructuredPatchFn is called with src (a copy of the original object
// converted to unstructured format) and dest (the object fetched from the
// API server).
//
// It returns an error if it fails to fetch the object, apply modifications,
// patch the object, or convert the result back to its typed form.
func PatchUnstructured(ctx context.Context, c client.Client, obj ObjectWithKind, modify UnstructuredPatchFn) error {
	destObj := unstructured.Unstructured{}
	destObj.SetGroupVersionKind(obj.GroupVersionKind())
	if err := c.Get(ctx, client.ObjectKeyFromObject(obj), &destObj); err != nil {
		return fmt.Errorf(
			"unable to get unstructured object for %s %q in namespace %q: %w",
			destObj.GroupVersionKind().Kind, obj.GetName(), obj.GetNamespace(), err,
		)
	}

	// Create a patch for the unstructured object.
	//
	// As we expect the object to be modified by the callback, while it may
	// also simultaneously be modified by other clients (e.g. someone updating
	// the object via `kubectl`), we use an optimistic lock to ensure that we
	// only apply the patch if the object has not been modified since we
	// fetched it.
	patch := client.MergeFromWithOptions(destObj.DeepCopy(), client.MergeFromWithOptimisticLock{})

	// Convert the typed object to an unstructured object.
	srcObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return fmt.Errorf("could not convert typed source object to unstructured object: %w", err)
	}
	srcApp := unstructured.Unstructured{Object: srcObj}

	// Apply modifications to the unstructured object.
	if err = modify(srcApp, destObj); err != nil {
		return fmt.Errorf("failed to apply modifications to unstructured object: %w", err)
	}

	// Issue the patch to the unstructured object.
	if err = c.Patch(ctx, &destObj, patch); err != nil {
		return fmt.Errorf("failed to patch the object: %w", err)
	}

	// Convert the unstructured object back to the typed object.
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(destObj.Object, obj); err != nil {
		return fmt.Errorf("error converting unstructured object to typed object: %w", err)
	}

	return nil
}
