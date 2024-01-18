package api

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
			return nil, nil, errors.Wrap(err, "error decoding manifest")
		}
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		resource := unstructured.Unstructured{}
		if err := yaml.Unmarshal(ext.Raw, &resource); err != nil {
			return nil, nil, errors.Wrap(err, "error unmarshaling manifest")
		}
		if (resource.GroupVersionKind().Group == kargoapi.GroupVersion.Group && resource.GetKind() == "Project") ||
			(resource.GroupVersionKind().Group == "" && resource.GetKind() == "Namespace") {
			projects = append(projects, resource)
		} else {
			otherResources = append(otherResources, resource)
		}
	}
	return projects, otherResources, nil
}
