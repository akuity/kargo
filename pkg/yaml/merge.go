package yaml

import (
	"errors"
	"fmt"
	"io"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/akuity/kargo/pkg/kustomize"
)

// Merge merges the provided YAML documents, in order, with the fields of each
// taking precedence over those of the documents before it. Lists are replaced
// rather than merged. Empty documents are skipped and the result is empty if
// every document is.
func Merge(docs [][]byte) ([]byte, error) {
	var mergeTarget *kyaml.RNode
	for i, doc := range docs {
		patchNode, err := kyaml.Parse(string(doc))
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			return nil, fmt.Errorf("error parsing document %d: %w", i, err)
		}
		if mergeTarget == nil {
			mergeTarget = patchNode
			continue
		}
		if mergeTarget, err = kustomize.Merge(
			patchNode, mergeTarget, kyaml.MergeOptions{},
		); err != nil {
			return nil, fmt.Errorf("error merging document %d: %w", i, err)
		}
	}
	if mergeTarget == nil {
		return []byte{}, nil
	}
	out, err := mergeTarget.String()
	if err != nil {
		return nil, fmt.Errorf("error serializing merged YAML: %w", err)
	}
	return []byte(out), nil
}
