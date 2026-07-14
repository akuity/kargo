package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/user"
)

const trueStr = "true"

// RefreshResourceType represents the type of Kargo resource to refresh. It is
// exported for use by the CLI, which uses it to identify the resource type
// requested by the user and to determine how to build the corresponding
// refresh request.
type RefreshResourceType string

// RefreshResourceType constants for supported resource types. They are
// PascalCase representations of the Kargo resource kinds for compatibility
// purposes with Kubernetes REST mappers.
const (
	RefreshResourceTypeClusterConfig RefreshResourceType = "ClusterConfig"
	RefreshResourceTypeProjectConfig RefreshResourceType = "ProjectConfig"
	RefreshResourceTypeStage         RefreshResourceType = "Stage"
	RefreshResourceTypeWarehouse     RefreshResourceType = "Warehouse"
)

// String returns the string representation of the RefreshResourceType.
func (t RefreshResourceType) String() string {
	return string(t)
}

// IsNamespaced returns true if the resource type is namespaced.
func (t RefreshResourceType) IsNamespaced() bool {
	return !strings.EqualFold(string(t), string(RefreshResourceTypeClusterConfig))
}

// NameEqualsProject returns true if the name of the resource should be the same
// as the project name. This is true for ProjectConfig resources.
func (t RefreshResourceType) NameEqualsProject() bool {
	return strings.EqualFold(string(t), string(RefreshResourceTypeProjectConfig))
}

var (
	projectGVK = schema.GroupVersionKind{
		Group:   kargoapi.GroupVersion.Group,
		Version: kargoapi.GroupVersion.Version,
		Kind:    "Project",
	}

	promotionGVK = schema.GroupVersionKind{
		Group:   kargoapi.GroupVersion.Group,
		Version: kargoapi.GroupVersion.Version,
		Kind:    "Promotion",
	}

	secretGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}

	freightGVK = schema.GroupVersionKind{
		Group:   kargoapi.GroupVersion.Group,
		Version: kargoapi.GroupVersion.Version,
		Kind:    "Freight",
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

// annotateResourceWithCreator annotates an unstructured object with information
// about the user who is creating the object, but only for resource types where
// that annotation is load-bearing -- i.e. where system behavior keys off of it.
// The API server creates resources using its own (control-plane) service
// account, so for those types, this annotation is the only record of the user
// on whose behalf it acted. The value set here overwrites anything in the
// caller's manifest, which also prevents callers from spoofing another
// identity. Types for which the annotation is purely informational are
// deliberately left untouched to avoid mutating user manifests unnecessarily.
func annotateResourceWithCreator(
	ctx context.Context,
	obj *unstructured.Unstructured,
) {
	if obj == nil {
		return
	}
	if gvk := obj.GroupVersionKind(); gvk != projectGVK && gvk != promotionGVK {
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
