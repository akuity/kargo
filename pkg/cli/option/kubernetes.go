package option

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

// ReadManifests reads Kubernetes manifests from local files or remote files
// via HTTP/S.
//
// WARNING: This function should not be used with untrusted input!
func ReadManifests(recursive bool, filenames ...string) ([]byte, error) {
	buildRes, err := resource.NewBuilder(&genericclioptions.ConfigFlags{}).
		Local().
		Unstructured().
		FilenameParam(false, &resource.FilenameOptions{
			Filenames: filenames,
			Recursive: recursive,
		}).
		Flatten().
		Do().
		Infos()
	if err != nil {
		return nil, fmt.Errorf("build resources: %w", err)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	defer func() {
		_ = enc.Close()
	}()
	for _, info := range buildRes {
		u, ok := info.Object.(*unstructured.Unstructured)
		if !ok {
			return nil, fmt.Errorf("expected *unstructured.Unstructured, got %T", info.Object)
		}
		if err := enc.Encode(&u.Object); err != nil {
			return nil, fmt.Errorf("encode object: %w", err)
		}
	}
	return buf.Bytes(), nil
}
