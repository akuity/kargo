package database

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
		row    Target
		assert func(*testing.T, kargoapi.Target, error)
	}{
		{
			name: "invalid labels",
			row:  Target{Name: "us-east", Labels: []byte(`[]`), Params: []byte(`{}`)},
			assert: func(t *testing.T, _ kargoapi.Target, err error) {
				require.ErrorContains(t, err, `error decoding labels of Target "us-east"`)
			},
		},
		{
			name: "invalid params",
			row:  Target{Name: "us-east", Labels: []byte(`{}`), Params: []byte(`nope`)},
			assert: func(t *testing.T, _ kargoapi.Target, err error) {
				require.ErrorContains(t, err, `error decoding params of Target "us-east"`)
			},
		},
		{
			name: "empty labels and params are absent from the resource",
			row: Target{
				ID: id, Name: "us-east", Labels: []byte(`{}`), Params: []byte(`{}`),
				CreatedAt: created, UpdatedAt: updated,
			},
			assert: func(t *testing.T, target kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Nil(t, target.Labels)
				require.Nil(t, target.Spec.Params)
			},
		},
		{
			name: "all fields",
			row: Target{
				ID:        id,
				Name:      "us-east",
				Labels:    []byte(`{"region":"us","tier":"prod"}`),
				Params:    []byte(`{"branch":"main","cluster":{"region":"us-east-1","replicas":3}}`),
				CreatedAt: created,
				UpdatedAt: updated,
			},
			assert: func(t *testing.T, target kargoapi.Target, err error) {
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
			target, err := TargetFromRow(testCase.row, "demo")
			testCase.assert(t, target, err)
		})
	}
}

func TestTargetsFromRows(t *testing.T) {
	t.Parallel()
	rows := []Target{
		{Name: "a", Labels: []byte(`{}`), Params: []byte(`{}`)},
		{Name: "b", Labels: []byte(`{"x":"y"}`), Params: []byte(`{}`)},
	}
	targets, err := TargetsFromRows(rows, "demo")
	require.NoError(t, err)
	require.Len(t, targets, 2)
	require.Equal(t, "a", targets[0].Name)
	require.Equal(t, map[string]string{"x": "y"}, targets[1].Labels)

	rows[1].Labels = []byte(`broken`)
	_, err = TargetsFromRows(rows, "demo")
	require.ErrorContains(t, err, `error decoding labels of Target "b"`)

	empty, err := TargetsFromRows(nil, "demo")
	require.NoError(t, err)
	require.NotNil(t, empty)
	require.Empty(t, empty)
}
