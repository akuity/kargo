package database

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestTargetFromRow(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := time.Date(2021, 6, 7, 8, 9, 10, 123456000, time.UTC)
	testCases := []struct {
		name   string
		row    TargetRow
		assert func(*testing.T, *kargoapi.Target, error)
	}{
		{
			name: "invalid params",
			row:  TargetRow{Name: "us-east", Params: json.RawMessage(`nope`)},
			assert: func(t *testing.T, _ *kargoapi.Target, err error) {
				require.ErrorContains(t, err, `error decoding params of Target "us-east"`)
			},
		},
		{
			name: "empty labels and params are absent from the resource",
			row: TargetRow{
				ID: id, Name: "us-east", Labels: labels.Set{}, Params: json.RawMessage(`{}`),
				CreatedAt: created, UpdatedAt: updated,
			},
			assert: func(t *testing.T, target *kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Nil(t, target.Labels)
				require.Nil(t, target.Spec.Params)
			},
		},
		{
			name: "all fields",
			row: TargetRow{
				ID:        id,
				Name:      "us-east",
				Labels:    labels.Set{"region": "us", "tier": "prod"},
				Params:    json.RawMessage(`{"branch":"main","cluster":{"region":"us-east-1","replicas":3}}`),
				CreatedAt: created,
				UpdatedAt: updated,
			},
			assert: func(t *testing.T, target *kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.GroupVersion.String(), target.APIVersion)
				require.Equal(t, "Target", target.Kind)
				require.Equal(t, "demo", target.Namespace)
				require.Equal(t, "us-east", target.Name)
				require.Equal(t, types.UID(id.String()), target.UID)
				require.Equal(t, "1623053350123456", target.ResourceVersion)
				require.True(t, target.CreationTimestamp.Time.Equal(created))
				require.Equal(t, map[string]string{"region": "us", "tier": "prod"}, target.Labels)
				require.Equal(t, map[string]apiextensionsv1.JSON{
					"branch":  {Raw: []byte(`"main"`)},
					"cluster": {Raw: []byte(`{"region":"us-east-1","replicas":3}`)},
				}, target.Spec.Params)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			target, err := targetFromRow(testCase.row, "demo")
			testCase.assert(t, target, err)
		})
	}
}

func TestTargetColumns(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		target *kargoapi.Target
		assert func(*testing.T, []byte, []byte, error)
	}{
		{
			name:   "nil maps become empty objects",
			target: &kargoapi.Target{},
			assert: func(t *testing.T, lbls, params []byte, err error) {
				require.NoError(t, err)
				require.JSONEq(t, `{}`, string(lbls))
				require.JSONEq(t, `{}`, string(params))
			},
		},
		{
			name: "values are kept as written",
			target: &kargoapi.Target{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"region": "us"},
				},
				Spec: kargoapi.TargetSpec{
					Params: map[string]apiextensionsv1.JSON{
						"cluster": {Raw: []byte(`{"replicas": 3}`)},
						"flag":    {Raw: []byte(`true`)},
					},
				},
			},
			assert: func(t *testing.T, lbls, params []byte, err error) {
				require.NoError(t, err)
				require.JSONEq(t, `{"region":"us"}`, string(lbls))
				require.JSONEq(t, `{"cluster":{"replicas":3},"flag":true}`, string(params))
			},
		},
		{
			name: "a param that is not JSON is rejected",
			target: &kargoapi.Target{
				ObjectMeta: metav1.ObjectMeta{Name: "bad"},
				Spec: kargoapi.TargetSpec{
					Params: map[string]apiextensionsv1.JSON{"x": {Raw: []byte(`nope`)}},
				},
			},
			assert: func(t *testing.T, _, _ []byte, err error) {
				require.ErrorContains(t, err, `error encoding params of Target "bad"`)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			lbls, params, err := targetColumns(testCase.target)
			testCase.assert(t, lbls, params, err)
		})
	}
}

func TestCheckTargetPreconditions(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	current := TargetRow{ID: id, UpdatedAt: time.UnixMicro(1623053350123456)}
	testCases := []struct {
		name   string
		target *kargoapi.Target
		assert func(*testing.T, error)
	}{
		{
			name:   "no preconditions",
			target: &kargoapi.Target{},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "matching preconditions",
			target: &kargoapi.Target{ObjectMeta: metav1.ObjectMeta{
				UID: types.UID(id.String()), ResourceVersion: "1623053350123456",
			}},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:   "a different UID",
			target: &kargoapi.Target{ObjectMeta: metav1.ObjectMeta{UID: "other"}},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, ErrConflict)
				require.ErrorContains(t, err, "recreated")
			},
		},
		{
			name:   "a stale resource version",
			target: &kargoapi.Target{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1"}},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, ErrConflict)
				require.ErrorContains(t, err, "modified")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assert(t, checkTargetPreconditions(testCase.target, current))
		})
	}
}

func TestTargetError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		err    error
		expect error
	}{
		{
			name:   "duplicate name",
			err:    &pgconn.PgError{Code: "23505", ConstraintName: targetNameConstraint},
			expect: ErrAlreadyExists,
		},
		{
			name:   "vanished project",
			err:    &pgconn.PgError{Code: "23503"},
			expect: ErrProjectNotMirrored,
		},
		{
			name:   "check constraint",
			err:    &pgconn.PgError{Code: "23514", Message: "labels"},
			expect: ErrInvalid,
		},
		{
			name:   "bad value",
			err:    &pgconn.PgError{Code: "22P05", Message: "unsupported Unicode escape"},
			expect: ErrInvalid,
		},
		{
			name:   "sentinel passes through",
			err:    ErrNotFound,
			expect: ErrNotFound,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := targetError(testCase.err, "demo", "us-east")
			require.ErrorIs(t, err, testCase.expect)
			require.ErrorContains(t, err, `Target "us-east" in Project "demo"`)
		})
	}
	other := errors.New("boom")
	err := targetError(other, "demo", "us-east")
	require.ErrorIs(t, err, other)
	require.NotErrorIs(t, err, ErrInvalid)
}
