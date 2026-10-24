package targets

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestTargetFromRow(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := time.Date(2021, 6, 7, 8, 9, 10, 123456000, time.UTC)
	testCases := []struct {
		name   string
		row    database.TargetRow
		assert func(*testing.T, *kargoapi.Target, error)
	}{
		{
			name: "invalid params",
			row:  database.TargetRow{Name: "us-east", Params: json.RawMessage(`nope`)},
			assert: func(t *testing.T, _ *kargoapi.Target, err error) {
				require.ErrorContains(t, err, `error decoding params of Target "us-east"`)
			},
		},
		{
			name: "empty labels and params are absent from the resource",
			row: database.TargetRow{
				ID:        id,
				Name:      "us-east",
				Labels:    map[string]string{},
				Params:    json.RawMessage(`{}`),
				CreatedAt: created,
				UpdatedAt: updated,
			},
			assert: func(t *testing.T, target *kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Nil(t, target.Labels)
				require.Nil(t, target.Spec.Params)
			},
		},
		{
			name: "all fields",
			row: database.TargetRow{
				ID:            id,
				Name:          "us-east",
				Labels:        map[string]string{"region": "us", "tier": "prod"},
				Params:        json.RawMessage(`{"branch":"main","cluster":{"region":"us-east-1","replicas":3}}`),
				TargetSetName: "fleet",
				CreatedAt:     created,
				UpdatedAt:     updated,
			},
			assert: func(t *testing.T, target *kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, "demo", target.Namespace)
				require.Equal(t, "us-east", target.Name)
				require.Equal(t, types.UID(id.String()), target.UID)
				require.Equal(t, "1623053350123456", target.ResourceVersion)
				require.True(t, target.CreationTimestamp.Time.Equal(created))
				require.Equal(t, map[string]string{"region": "us", "tier": "prod"}, target.Labels)
				require.Equal(t, "fleet", target.Spec.TargetSet)
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

func TestSetStatuses(t *testing.T) {
	t.Parallel()
	known := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	targets := []kargoapi.Target{{
		ObjectMeta: metav1.ObjectMeta{Name: "us-east", UID: types.UID(known.String())},
	}}
	row := func(id uuid.UUID, stage, warehouse, freight string) database.ListTargetCurrentFreightRow {
		return database.ListTargetCurrentFreightRow{
			TargetID:    id,
			StageName:   stage,
			OriginKind:  string(kargoapi.FreightOriginKindWarehouse),
			OriginName:  warehouse,
			FreightName: freight,
		}
	}
	setStatuses(targets, []database.ListTargetCurrentFreightRow{
		row(known, "canary", "w1", "c1"),
		row(known, "fleet", "w1", "f1"),
		row(known, "fleet", "w2", "f2"),
		row(uuid.New(), "fleet", "w1", "other"),
	})
	stages := targets[0].Status.Stages
	require.Len(t, stages, 2)
	// The collection is the one a Stage would build from the same Freight, ID
	// and all.
	want := &kargoapi.FreightCollection{}
	want.UpdateOrPush(
		kargoapi.FreightReference{
			Name:   "f1",
			Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "w1"},
		},
		kargoapi.FreightReference{
			Name:   "f2",
			Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "w2"},
		},
	)
	require.Equal(t, want, stages["fleet"].CurrentFreight)
	require.Nil(t, stages["fleet"].Health)
	require.Nil(t, stages["fleet"].HealthChecks)
	require.Len(t, stages["canary"].CurrentFreight.Freight, 1)
	require.Equal(t, "c1", stages["canary"].CurrentFreight.Freight["Warehouse/w1"].Name)
}

func TestRowFromTarget(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		target *kargoapi.Target
		assert func(*testing.T, database.TargetRow, error)
	}{
		{
			name:   "absent labels and params stay absent",
			target: &kargoapi.Target{ObjectMeta: metav1.ObjectMeta{Name: "us-east"}},
			assert: func(t *testing.T, row database.TargetRow, err error) {
				require.NoError(t, err)
				require.Equal(t, "us-east", row.Name)
				require.Nil(t, row.Labels)
				require.Nil(t, row.Params)
			},
		},
		{
			name: "only name, labels, params and TargetSet are taken",
			target: &kargoapi.Target{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "us-east",
					UID:             "client-chosen",
					ResourceVersion: "999",
					Labels:          map[string]string{"region": "us"},
				},
				Spec: kargoapi.TargetSpec{
					TargetSet: "fleet",
					Params: map[string]apiextensionsv1.JSON{
						"cluster": {Raw: []byte(`{"replicas": 3}`)},
						"flag":    {Raw: []byte(`true`)},
					},
				},
			},
			assert: func(t *testing.T, row database.TargetRow, err error) {
				require.NoError(t, err)
				require.Equal(t, map[string]string{"region": "us"}, row.Labels)
				require.JSONEq(t, `{"cluster":{"replicas":3},"flag":true}`, string(row.Params))
				require.Equal(t, "fleet", row.TargetSetName)
				require.Equal(t, uuid.Nil, row.ID)
				require.True(t, row.UpdatedAt.IsZero())
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
			assert: func(t *testing.T, _ database.TargetRow, err error) {
				require.ErrorContains(t, err, `error encoding params of Target "bad"`)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			row, err := rowFromTarget(testCase.target)
			testCase.assert(t, row, err)
		})
	}
}

func TestSetPreconditions(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	testCases := []struct {
		name   string
		meta   metav1.ObjectMeta
		assert func(*testing.T, database.TargetRow, error)
	}{
		{
			name: "none given",
			assert: func(t *testing.T, row database.TargetRow, err error) {
				require.NoError(t, err)
				require.Equal(t, uuid.Nil, row.ID)
				require.True(t, row.UpdatedAt.IsZero())
			},
		},
		{
			name: "a resource version round trips to updated_at",
			meta: metav1.ObjectMeta{
				UID:             types.UID(id.String()),
				ResourceVersion: "1623053350123456",
			},
			assert: func(t *testing.T, row database.TargetRow, err error) {
				require.NoError(t, err)
				require.Equal(t, id, row.ID)
				require.Equal(t, "1623053350123456", resourceVersion(row.UpdatedAt))
			},
		},
		{
			name: "a UID the server did not issue is a conflict",
			meta: metav1.ObjectMeta{UID: "not-a-uuid"},
			assert: func(t *testing.T, _ database.TargetRow, err error) {
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "recreated")
			},
		},
		{
			name: "a resource version the server did not issue is a conflict",
			meta: metav1.ObjectMeta{ResourceVersion: "stale"},
			assert: func(t *testing.T, _ database.TargetRow, err error) {
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "modified")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var row database.TargetRow
			err := setPreconditions(&kargoapi.Target{ObjectMeta: testCase.meta}, &row)
			testCase.assert(t, row, err)
		})
	}
}
