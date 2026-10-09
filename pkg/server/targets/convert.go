package targets

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

// targetFromRow presents a row as the Target resource the API serves. The
// row's id is the UID and its last update, which every write advances, the
// resource version.
func targetFromRow(row database.TargetRow, project string) (*kargoapi.Target, error) {
	var rawParams map[string]json.RawMessage
	if err := json.Unmarshal(row.Params, &rawParams); err != nil {
		return nil, fmt.Errorf("error decoding params of Target %q: %w", row.Name, err)
	}
	var params map[string]apiextensionsv1.JSON
	if len(rawParams) > 0 {
		params = make(map[string]apiextensionsv1.JSON, len(rawParams))
		for key, raw := range rawParams {
			params[key] = apiextensionsv1.JSON{Raw: raw}
		}
	}
	var lbls map[string]string
	if len(row.Labels) > 0 {
		lbls = row.Labels
	}
	return &kargoapi.Target{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Target",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         project,
			Name:              row.Name,
			UID:               types.UID(row.ID.String()),
			ResourceVersion:   resourceVersion(row.UpdatedAt),
			CreationTimestamp: metav1.NewTime(row.CreatedAt),
			Labels:            lbls,
		},
		Spec: kargoapi.TargetSpec{Params: params},
	}, nil
}

// rowFromTarget takes what the database stores from a Target: its name,
// labels and params. Everything else about the resource is the database's to
// assign.
func rowFromTarget(target *kargoapi.Target) (database.TargetRow, error) {
	row := database.TargetRow{
		Name:   target.Name,
		Labels: target.Labels,
	}
	if len(target.Spec.Params) > 0 {
		params := make(map[string]json.RawMessage, len(target.Spec.Params))
		for key, value := range target.Spec.Params {
			params[key] = json.RawMessage(value.Raw)
		}
		var err error
		if row.Params, err = json.Marshal(params); err != nil {
			return database.TargetRow{}, fmt.Errorf(
				"error encoding params of Target %q: %w", target.Name, err,
			)
		}
	}
	return row, nil
}

// setPreconditions carries a Target's UID and resource version, if given,
// onto the row as the identity and version an update requires. A value that
// is not one the server issued can match no Target, so it is a conflict.
func setPreconditions(target *kargoapi.Target, row *database.TargetRow) error {
	if target.UID != "" {
		id, err := uuid.Parse(string(target.UID))
		if err != nil {
			return fmt.Errorf("%w: the Target was deleted and recreated", database.ErrConflict)
		}
		row.ID = id
	}
	if target.ResourceVersion != "" {
		micros, err := strconv.ParseInt(target.ResourceVersion, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: the Target has been modified", database.ErrConflict)
		}
		row.UpdatedAt = time.UnixMicro(micros)
	}
	return nil
}

// resourceVersion derives a Target's resource version from its row's last
// update, which the schema guarantees grows with every write to the row.
func resourceVersion(updatedAt time.Time) string {
	return strconv.FormatInt(updatedAt.UnixMicro(), 10)
}
