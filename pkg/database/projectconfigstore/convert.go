package projectconfigstore

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

// ToAPI presents a stored ProjectConfig as the resource the API serves. Its
// name and namespace are both the Project's name, the row's id is the UID and
// its last update, which every write advances, the resource version. A
// requested refresh is served as the refresh annotation as well as from the
// row's own column, so that clients which wait on the annotation keep
// working. The webhook receivers are served from their own rows, ordered by
// name.
//
// The spec and conditions are decoded from what the store wrote from the
// same types, so they always decode; a column that somehow does not is
// served as empty rather than failing the read.
func ToAPI(cfg Config) *kargoapi.ProjectConfig {
	out := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              cfg.ProjectName,
			Namespace:         cfg.ProjectName,
			UID:               types.UID(cfg.ID.String()),
			ResourceVersion:   resourceVersion(cfg.UpdatedAt),
			Generation:        cfg.Generation,
			CreationTimestamp: metav1.NewTime(cfg.CreatedAt),
			Labels:            mapOrNil(cfg.Labels),
			Annotations:       mapOrNil(cfg.Annotations),
		},
	}
	if cfg.RefreshToken != nil {
		annotations := maps.Clone(cfg.Annotations)
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[kargoapi.AnnotationKeyRefresh] = *cfg.RefreshToken
		out.Annotations = annotations
	}
	if len(cfg.Spec) > 0 {
		_ = json.Unmarshal(cfg.Spec, &out.Spec)
	}
	if len(cfg.StatusConditions) > 0 {
		_ = json.Unmarshal(cfg.StatusConditions, &out.Status.Conditions)
	}
	if cfg.StatusObservedGeneration != nil {
		out.Status.ObservedGeneration = *cfg.StatusObservedGeneration
	}
	if cfg.StatusLastHandledRefresh != nil {
		out.Status.LastHandledRefresh = *cfg.StatusLastHandledRefresh
	}
	out.Status.WebhookReceivers = receiverDetails(cfg.Receivers)
	return out
}

// receiverDetails presents receiver rows as status.webhookReceivers, ordered
// by name. No rows are presented as none.
func receiverDetails(
	rows []database.ProjectConfigWebhookReceiverRow,
) []kargoapi.WebhookReceiverDetails {
	if len(rows) == 0 {
		return nil
	}
	details := make([]kargoapi.WebhookReceiverDetails, len(rows))
	for i, row := range rows {
		details[i] = kargoapi.WebhookReceiverDetails{
			Name: row.Name,
			Path: row.Path,
			URL:  row.URL,
		}
	}
	slices.SortFunc(details, func(a, b kargoapi.WebhookReceiverDetails) int {
		return strings.Compare(a.Name, b.Name)
	})
	return details
}

// specColumns encodes what the database stores of a ProjectConfig's metadata
// and spec. The spec is encoded from its type, so fields the type does not
// know are dropped. Labels and annotations are kept as written, except the
// refresh annotation: a refresh is requested through RequestRefresh, and a
// client that read the config and writes it back must not store the token it
// was served. Absent maps become empty objects, since the columns are NOT
// NULL. Nothing else of the metadata, such as finalizers or owner
// references, is stored.
func specColumns(cfg *kargoapi.ProjectConfig) (labels, annotations, spec []byte, err error) {
	labelMap := cfg.Labels
	if labelMap == nil {
		labelMap = map[string]string{}
	}
	if labels, err = json.Marshal(labelMap); err != nil {
		return nil, nil, nil, fmt.Errorf("error encoding labels: %w", err)
	}
	annotationMap := maps.Clone(cfg.Annotations)
	if annotationMap == nil {
		annotationMap = map[string]string{}
	}
	delete(annotationMap, kargoapi.AnnotationKeyRefresh)
	if annotations, err = json.Marshal(annotationMap); err != nil {
		return nil, nil, nil, fmt.Errorf("error encoding annotations: %w", err)
	}
	if spec, err = json.Marshal(cfg.Spec); err != nil {
		return nil, nil, nil, fmt.Errorf("error encoding spec: %w", err)
	}
	return labels, annotations, spec, nil
}

// statusConditions encodes conditions for the database. None are stored as
// NULL.
func statusConditions(conditions []metav1.Condition) ([]byte, error) {
	if len(conditions) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(conditions)
	if err != nil {
		return nil, fmt.Errorf("error encoding conditions: %w", err)
	}
	return data, nil
}

// resourceVersion renders a row's last update as the resource version of
// the object the API serves: microseconds since the epoch, which the
// database keeps unique per row. It is the encoding Targets use.
func resourceVersion(updatedAt time.Time) string {
	return strconv.FormatInt(updatedAt.UnixMicro(), 10)
}

// parseResourceVersion reads a resource version rendered by resourceVersion
// back into the last update it stands for. A value it did not render is
// database.ErrInvalid.
func parseResourceVersion(rv string) (time.Time, error) {
	micros, err := strconv.ParseInt(rv, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("resource version %q: %w", rv, database.ErrInvalid)
	}
	return time.UnixMicro(micros), nil
}

func mapOrNil(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	return maps.Clone(m)
}
