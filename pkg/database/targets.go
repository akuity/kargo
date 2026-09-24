package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// ErrNotFound is returned when a row that was asked for by name does not
// exist.
var ErrNotFound = errors.New("not found")

// ErrNotMirrored is returned when a write refers, by name, to a Project,
// Stage, Freight or Target that has no row. Kubernetes resources reach the
// database with some lag, so a caller that has just seen the resource in
// Kubernetes should retry.
var ErrNotMirrored = errors.New("referenced resource is not in the database")

func (s *store) ListTargets(ctx context.Context, projectName string) ([]Target, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.ListTargets(ctx, projectName)
}

func (s *store) GetTarget(ctx context.Context, params GetTargetParams) (Target, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	target, err := s.queries.GetTarget(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return Target{}, fmt.Errorf("Target %q in Project %q: %w", params.Name, params.ProjectName, ErrNotFound)
	}
	return target, err
}

// TargetFromRow presents a Target row in the shape of the Target resource, so
// that code written against the resource can consume rows unchanged. The row's
// id becomes the UID and its last update becomes the resource version.
func TargetFromRow(row Target, projectName string) (kargoapi.Target, error) {
	var labels map[string]string
	if err := json.Unmarshal(row.Labels, &labels); err != nil {
		return kargoapi.Target{}, fmt.Errorf("error decoding labels of Target %q: %w", row.Name, err)
	}
	if len(labels) == 0 {
		labels = nil
	}
	var rawParams map[string]json.RawMessage
	if err := json.Unmarshal(row.Params, &rawParams); err != nil {
		return kargoapi.Target{}, fmt.Errorf("error decoding params of Target %q: %w", row.Name, err)
	}
	var params map[string]apiextensionsv1.JSON
	if len(rawParams) > 0 {
		params = make(map[string]apiextensionsv1.JSON, len(rawParams))
		for key, raw := range rawParams {
			params[key] = apiextensionsv1.JSON{Raw: raw}
		}
	}
	return kargoapi.Target{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Target",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         projectName,
			Name:              row.Name,
			UID:               types.UID(row.ID.String()),
			ResourceVersion:   resourceVersion(row.UpdatedAt),
			CreationTimestamp: metav1.NewTime(row.CreatedAt),
			Labels:            labels,
		},
		Spec: kargoapi.TargetSpec{Params: params},
	}, nil
}

// TargetsFromRows converts every row with TargetFromRow.
func TargetsFromRows(rows []Target, projectName string) ([]kargoapi.Target, error) {
	targets := make([]kargoapi.Target, len(rows))
	for i, row := range rows {
		target, err := TargetFromRow(row, projectName)
		if err != nil {
			return nil, err
		}
		targets[i] = target
	}
	return targets, nil
}

// resourceVersion derives a resource version from a row's last update. It
// only has to grow whenever the row changes, which the update time does at
// microsecond resolution.
func resourceVersion(updatedAt time.Time) string {
	return strconv.FormatInt(updatedAt.UnixMicro(), 10)
}
