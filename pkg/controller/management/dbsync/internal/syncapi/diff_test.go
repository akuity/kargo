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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestDiff(t *testing.T) {
	t.Parallel()
	req := func(name string) reconcile.Request {
		return reconcile.Request{NamespacedName: client.ObjectKey{Name: name}}
	}
	testCases := []struct {
		name      string
		rows      map[string]string // UID -> name
		matchErr  error
		wrongType bool
		want      []reconcile.Request
		wantErr   string
	}{
		{
			// a: no row. b: row differs. c: matches. Two orphaned rows whose
			// objects are gone come last, sorted by name.
			name: "only missing, differing, and orphaned objects", rows: map[string]string{
				"b-uid": "outdated", "c-uid": "c", "z-stale": "z", "a-stale": "old",
			},
			want: []reconcile.Request{req("a"), req("b"), req("old"), req("z")},
		},
		{
			name: "all rows match", rows: map[string]string{"a-uid": "a", "b-uid": "b", "c-uid": "c"},
		},
		{
			// The row for "a" carries an old UID, so "a" is both a live object
			// without a matching row and the name of an orphaned row. It is
			// returned once.
			name: "recreated object is returned once", rows: map[string]string{
				"a-old": "a", "b-uid": "b", "c-uid": "c",
			},
			want: []reconcile.Request{req("a")},
		},
		{
			name: "comparison error discards partial results", rows: map[string]string{"b-uid": "b", "old": "old"},
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
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "a", UID: "a-uid"}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "b", UID: "b-uid"}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "c", UID: "c-uid"}},
				&kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "dev", UID: "stage"}},
			).Build()
			var list client.ObjectList = &kargoapi.ProjectList{}
			if testCase.wrongType {
				list = &kargoapi.StageList{}
			}
			snapshot := maps.Clone(testCase.rows)
			requests, err := Diff(
				context.Background(),
				kube,
				list,
				snapshot,
				func(obj *kargoapi.Project, name string) (bool, error) {
					return obj.Name == name, testCase.matchErr
				},
				func(name string) client.ObjectKey { return client.ObjectKey{Name: name} },
			)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.want, requests)
			require.Equal(t, testCase.rows, snapshot)
		})
	}
}
