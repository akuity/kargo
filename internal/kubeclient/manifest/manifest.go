package manifest

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
)

type ParseFunc func(data []byte) ([]*unstructured.Unstructured, error)

func NewParser(scheme *runtime.Scheme) ParseFunc {
	codecs := serializer.NewCodecFactory(scheme)
	deserializer := codecs.UniversalDeserializer()
	return func(data []byte) ([]*unstructured.Unstructured, error) {
		d := kubeyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
		var res []*unstructured.Unstructured
		for {
			var ext runtime.RawExtension
			if err := d.Decode(&ext); err != nil {
				if errors.Is(err, io.EOF) {
					return res, nil
				}
				return nil, errors.Wrap(err, "decode data")
			}

			obj, _, err := deserializer.Decode(ext.Raw, nil, nil)
			if err != nil {
				return nil, errors.Wrap(err, "decode object")
			}
			u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
			if err != nil {
				return nil, errors.Wrap(err, "convert to unstructured")
			}
			res = append(res, &unstructured.Unstructured{Object: u})
		}
	}
}
