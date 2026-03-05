package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/user"
)

const trueStr = "true"

var (
	projectGVK = schema.GroupVersionKind{
		Group:   kargoapi.GroupVersion.Group,
		Version: kargoapi.GroupVersion.Version,
		Kind:    "Project",
	}

	secretGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}

	errSecretManagementDisabled = libhttp.ErrorStr(
		"secret management is not enabled",
		http.StatusNotImplemented,
	)

	errArgoRolloutsIntegrationDisabled = libhttp.ErrorStr(
		"Argo Rollouts integration is not enabled",
		http.StatusNotImplemented,
	)

	errEmptySecret = libhttp.ErrorStr(
		"cannot have empty secret",
		http.StatusBadRequest,
	)
)

// splitYAML splits YAML bytes into unstructured objects. It separates Project
// and Namespace resources from all other resources and returns them separately.
// This is because Project and Namespace commonly need to be created first and
// deleted last. This is adapted from GitOps Engine.
func splitYAML(
	yamlData []byte,
) ([]unstructured.Unstructured, []unstructured.Unstructured, error) {
	trimmed := bytes.TrimSpace(yamlData)

	// If input is empty or whitespace-only, return empty results
	if len(trimmed) == 0 {
		return nil, nil, nil
	}

	// If input starts with '[', it's a JSON array - handle separately
	if trimmed[0] == '[' {
		return splitJSONArray(trimmed)
	}

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlData), 4096)
	var projects, otherResources []unstructured.Unstructured
	for {
		ext := runtime.RawExtension{}
		if err := decoder.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, fmt.Errorf("error decoding manifest: %w", err)
		}
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		resource := unstructured.Unstructured{}
		if err := yaml.Unmarshal(ext.Raw, &resource); err != nil {
			return nil, nil, fmt.Errorf("error unmarshaling manifest: %w", err)
		}
		if resource.GroupVersionKind() == projectGVK {
			projects = append(projects, resource)
		} else {
			otherResources = append(otherResources, resource)
		}
	}
	return projects, otherResources, nil
}

// splitJSONArray handles JSON array input, splitting it into individual resources.
func splitJSONArray(
	data []byte,
) ([]unstructured.Unstructured, []unstructured.Unstructured, error) {
	var rawObjects []json.RawMessage
	if err := json.Unmarshal(data, &rawObjects); err != nil {
		return nil, nil, fmt.Errorf("error decoding JSON array: %w", err)
	}

	var projects, otherResources []unstructured.Unstructured
	for _, raw := range rawObjects {
		resource := unstructured.Unstructured{}
		if err := json.Unmarshal(raw, &resource.Object); err != nil {
			return nil, nil, fmt.Errorf("error unmarshaling JSON object: %w", err)
		}
		if resource.GroupVersionKind() == projectGVK {
			projects = append(projects, resource)
		} else {
			otherResources = append(otherResources, resource)
		}
	}
	return projects, otherResources, nil
}

// objectOrRaw takes structured or unstructured objects as input and depending
// on requested format returns EITHER (but never both) the object serialized in
// the requested format OR the object converted to the structured object type.
func objectOrRaw[T client.Object](
	c client.Client,
	obj client.Object,
	format svcv1alpha1.RawFormat,
	t T,
) (T, []byte, error) {
	if _, ok := obj.(*unstructured.Unstructured); !ok {
		// Structured objects are likely to be missing GVK information, so we add
		// it in.
		gvk, err := c.GroupVersionKindFor(t)
		if err != nil {
			return *new(T), nil,
				fmt.Errorf("could not determine GVK for type: %w", err)
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
	}
	switch format {
	case svcv1alpha1.RawFormat_RAW_FORMAT_JSON:
		raw, err := json.Marshal(obj)
		if err != nil {
			return *new(T), nil,
				fmt.Errorf("object could not be marshaled to raw JSON: %w", err)
		}
		return *new(T), raw, nil
	case svcv1alpha1.RawFormat_RAW_FORMAT_YAML:
		raw, err := sigyaml.Marshal(obj)
		if err != nil {
			return *new(T), nil,
				fmt.Errorf("object could not be marshaled to raw YAML: %w", err)
		}
		return *new(T), raw, nil
	}
	if uObj, ok := obj.(*unstructured.Unstructured); ok {
		var newObj T
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uObj.Object, &newObj); err != nil {
			return *new(T), nil, fmt.Errorf(
				"error converting unstructured object to typed object: %w", err,
			)
		}
		return newObj, nil, nil
	}
	if typed, ok := obj.(T); ok {
		return typed, nil, nil
	}
	return *new(T), nil,
		fmt.Errorf("type mismatch: cannot input to expected type")
}

// annotateProjectWithCreator annotates an unstructured object with information
// about the user who is creating the object only if that unstructured object
// represents a Project.
func annotateProjectWithCreator(
	ctx context.Context,
	obj *unstructured.Unstructured,
) {
	if obj == nil || obj.GroupVersionKind() != projectGVK {
		return
	}
	if userInfo, found := user.InfoFromContext(ctx); found {
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[kargoapi.AnnotationKeyCreateActor] = api.FormatEventUserActor(userInfo)
		obj.SetAnnotations(annotations)
	}
}
