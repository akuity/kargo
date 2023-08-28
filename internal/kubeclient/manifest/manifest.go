package manifest

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/kustomize/api/provider"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
)

type ParseFunc func(data []byte) (cluster, namespaced []*unstructured.Unstructured, err error)

// NewParser returns a new parser that parses Kubernetes manifest and
// returns parsed objects in cluster - namespaced order.
func NewParser(scheme *runtime.Scheme) ParseFunc {
	codecs := serializer.NewCodecFactory(scheme)
	deserializer := codecs.UniversalDeserializer()
	resourceFactory := provider.NewDefaultDepProvider().GetResourceFactory()
	factory := resmap.NewFactory(resourceFactory)

	return func(data []byte) (cluster, namespaced []*unstructured.Unstructured, err error) {
		resMap, err := factory.NewResMapFromBytes(data)
		if err != nil {
			return nil, nil, errors.Wrap(err, "new resmap from data")
		}
		cluster = make([]*unstructured.Unstructured, 0, resMap.Size())
		for _, r := range resMap.ClusterScoped() {
			u, err := resourceToUnstructured(deserializer, r)
			if err != nil {
				return nil, nil, errors.Wrap(err, "resource to unstructured")
			}
			cluster = append(cluster, u)
		}
		namespaced = make([]*unstructured.Unstructured, 0, resMap.Size()-len(cluster))
		for _, resources := range resMap.GroupedByOriginalNamespace() {
			for _, r := range resources {
				u, err := resourceToUnstructured(deserializer, r)
				if err != nil {
					return nil, nil, errors.Wrap(err, "resource to unstructured")
				}
				namespaced = append(namespaced, u)
			}
		}
		return cluster, namespaced, nil
	}
}

func resourceToUnstructured(decoder runtime.Decoder, r *resource.Resource) (*unstructured.Unstructured, error) {
	data, err := r.AsYAML()
	if err != nil {
		return nil, errors.Wrap(err, "resource to yaml")
	}
	obj, _, err := decoder.Decode(data, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "decode object")
	}
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
	if err != nil {
		return nil, errors.Wrap(err, "convert to unstructured")
	}
	return &unstructured.Unstructured{Object: u}, nil
}
