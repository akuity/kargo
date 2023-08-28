package kubeclient

import (
	"context"
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch/v5"
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
