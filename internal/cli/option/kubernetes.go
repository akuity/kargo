package option

import (
	"bytes"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func ReadManifests(filenames ...string) ([]byte, error) {
	buildRes, err := cmdutil.NewFactory(&genericclioptions.ConfigFlags{}).
		NewBuilder().
		Unstructured().
		FilenameParam(false, &resource.FilenameOptions{
			Filenames: filenames,
			Recursive: false,
		}).
		Flatten().
		Do().
		Infos()
	if err != nil {
		return nil, errors.Wrap(err, "build resources")
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	defer func() {
		_ = enc.Close()
	}()
	for _, info := range buildRes {
		u, ok := info.Object.(*unstructured.Unstructured)
		if !ok {
			return nil, errors.Errorf("expected *unstructured.Unstructured, got %T", info.Object)
		}
		if err := enc.Encode(&u.Object); err != nil {
			return nil, errors.Wrap(err, "encode object")
		}
	}
	return buf.Bytes(), nil
}
