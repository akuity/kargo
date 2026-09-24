package syncapi

import (
	"context"
	"errors"
	"maps"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestDiff(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		rows       map[string]string
		matchErr   error
		wrongType  bool
		wantSync   []client.ObjectKey
		wantDelete []string
		wantErr    string
	}{
		{
			name: "only missing and differing objects", rows: map[string]string{
				"b-uid": "outdated", "c-uid": "c", "z-stale": "z", "a-stale": "old",
			},
			wantSync:   []client.ObjectKey{{Namespace: "demo", Name: "a"}, {Namespace: "demo", Name: "b"}},
			wantDelete: []string{"a-stale", "z-stale"},
		},
		{
			name: "all rows match", rows: map[string]string{"a-uid": "a", "b-uid": "b", "c-uid": "c"},
		},
		{
			name: "comparison error discards partial changes", rows: map[string]string{"b-uid": "b", "old": "old"},
			matchErr: errors.New("conversion failed"), wantErr: "conversion failed",
		},
		{name: "wrong list item type", wrongType: true, wantErr: "unexpected object"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "a", UID: "a-uid"}},
				&kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "b", UID: "b-uid"}},
				&kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "c", UID: "c-uid"}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "demo", UID: "parent"}},
			).Build()
			var list client.ObjectList = &kargoapi.StageList{}
			if testCase.wrongType {
				list = &kargoapi.ProjectList{}
			}
			snapshot := maps.Clone(testCase.rows)
			changes, err := Diff(
				context.Background(),
				kube,
				list,
				snapshot,
				func(obj *kargoapi.Stage, name string) (bool, error) {
					return obj.Name == name, testCase.matchErr
				},
			)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, Changes{ToSync: testCase.wantSync, ToDelete: testCase.wantDelete}, changes)
			require.Equal(t, testCase.rows, snapshot)
		})
	}
}
