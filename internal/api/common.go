package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigyaml "sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

var (
	secretGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}

	errSecretManagementDisabled = fmt.Errorf("secret management is not enabled")
)

// splitYAML splits YAML bytes into unstructured objects. It separates Project
// and Namespace resources from all other resources and returns them separately.
// This is because Project and Namespace commonly need to be created first and
// deleted last. This is adapted from GitOps Engine.
func splitYAML(
	yamlData []byte,
) ([]unstructured.Unstructured, []unstructured.Unstructured, error) {
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
		if resource.GroupVersionKind().Group == kargoapi.GroupVersion.Group && resource.GetKind() == "Project" {
			projects = append(projects, resource)
		} else {
			otherResources = append(otherResources, resource)
		}
	}
	return projects, otherResources, nil
}

// objectOrRaw returns either the object or the raw representation of the object
// based on the format.
func objectOrRaw[T client.Object](obj T, format svcv1alpha1.RawFormat) (T, []byte, error) {
	switch format {
	case svcv1alpha1.RawFormat_RAW_FORMAT_JSON:
		raw, err := json.Marshal(obj)
		if err != nil {
			return *new(T), nil, fmt.Errorf("object could not be marshaled to raw JSON: %w", err)
		}
		return *new(T), raw, nil
	case svcv1alpha1.RawFormat_RAW_FORMAT_YAML:
		raw, err := sigyaml.Marshal(obj)
		if err != nil {
			return *new(T), nil, fmt.Errorf("object could not be marshaled to raw YAML: %w", err)
		}
		return *new(T), raw, nil
	default:
		return obj, nil, nil
	}
}
