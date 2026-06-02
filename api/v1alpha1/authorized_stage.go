package v1alpha1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
)

// AuthorizedStages parses the value of an AnnotationKeyAuthorizedStage
// annotation into the set of Stages authorized to manage the annotated
// resource (typically an Argo CD Application).
//
// The value is a comma-separated list of "<project>:<stage>" entries,
// permitting more than one Stage to be authorized. A single "<project>:<stage>"
// value (the historical format) parses to a single-element list. The returned
// NamespacedNames use the project as the Namespace and the Stage name as the
// Name.
//
// Whitespace around entries and around the project and Stage of each entry is
// ignored. Deprecated glob expressions (entries containing "*") are rejected
// with an error, as are malformed or empty entries.
func AuthorizedStages(value string) ([]types.NamespacedName, error) {
	var stages []types.NamespacedName
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		project, stage, ok := strings.Cut(entry, ":")
		project, stage = strings.TrimSpace(project), strings.TrimSpace(stage)
		if !ok || project == "" || stage == "" {
			return nil, fmt.Errorf(
				"invalid authorized Stage %q: expected format %q",
				entry, "<project>:<stage>",
			)
		}
		if strings.Contains(project, "*") || strings.Contains(stage, "*") {
			return nil, fmt.Errorf(
				"invalid authorized Stage %q: deprecated glob expressions are no longer supported",
				entry,
			)
		}
		stages = append(stages, types.NamespacedName{Namespace: project, Name: stage})
	}
	if len(stages) == 0 {
		return nil, fmt.Errorf("no authorized Stages found in %q", value)
	}
	return stages, nil
}
