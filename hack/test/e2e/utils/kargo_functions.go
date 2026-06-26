//nolint:forcetypeassert
package utils

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

func UpdatePromotionTasksVar(name, key, val string) decoder.DecodeOption {
	return MutateAsUnstructuredOptionFor("PromotionTask", name, func(unstr runtime.Unstructured) error {
		data := unstr.UnstructuredContent()
		fmt.Printf("Parsed data %v\n", data)
		for _, tplVar := range data["spec"].(map[string]any)["vars"].([]any) {
			tplVarMap := tplVar.(map[string]any)
			if tplVarMap["name"] == key {
				tplVarMap["value"] = val
			}
		}

		fmt.Printf("Updated data %v\n", data)

		unstr.SetUnstructuredContent(data)
		return nil
	})
}

func UpdateWarehouseGitRepoURL(name, repoURL string) decoder.DecodeOption {
	return MutateAsUnstructuredOptionFor("Warehouse", name, func(unstr runtime.Unstructured) error {
		data := unstr.UnstructuredContent()
		fmt.Printf("Parsed data %v\n", data)
		for _, sub := range data["spec"].(map[string]any)["subscriptions"].([]any) {
			subMap := sub.(map[string]any)

			if gitSub, ok := subMap["git"]; ok {
				gitSubMap := gitSub.(map[string]any)
				gitSubMap["repoURL"] = repoURL
			}
		}

		fmt.Printf("Updated data %v\n", data)

		unstr.SetUnstructuredContent(data)
		return nil
	})
}

func MutateAsUnstructuredOptionFor(
	kind, name string,
	mutateFunc func(obj runtime.Unstructured) error,
) decoder.DecodeOption {
	return MutateOptionFor(kind, name, func(obj k8s.Object) error {
		return MutateAsUnstructured(obj, mutateFunc)
	})
}

func MutateAsUnstructured(obj k8s.Object, mutateFunc func(obj runtime.Unstructured) error) error {
	if unstr, ok := obj.(runtime.Unstructured); ok {
		return mutateFunc(unstr)
	}
	return errors.New("object is not Unstructured")
}

func MutateOptionFor(kind, name string, mutateFunc func(obj k8s.Object) error) decoder.DecodeOption {
	return decoder.MutateOption(func(obj k8s.Object) error {
		if obj.GetObjectKind().GroupVersionKind().Kind == kind {
			if name == "" || obj.GetName() == name {
				return mutateFunc(obj)
			}
		}
		return nil
	})
}
