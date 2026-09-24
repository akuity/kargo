package freight

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestDiffArtifacts(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	parent := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "demo", UID: "project"}}
	freight := testFreight("demo", "freight", created)
	for _, subscription := range []string{"first", "second"} {
		freight.Commits = append(freight.Commits, kargoapi.GitCommit{
			RepoURL: "https://example.com/repo", ID: "abc", Tag: "v1", Branch: "main",
			Author: "author", Committer: "committer", Message: "message", SubscriptionName: subscription,
		})
		freight.Images = append(freight.Images, kargoapi.Image{
			RepoURL: "example.com/image", Tag: "v1", Digest: "sha256:abc", SubscriptionName: subscription,
			Annotations: map[string]string{"build": "123"},
		})
		freight.Charts = append(freight.Charts, kargoapi.Chart{
			RepoURL: "https://example.com/charts", Name: "app", Version: "1.0", SubscriptionName: subscription,
		})
		freight.Artifacts = append(freight.Artifacts, kargoapi.ArtifactReference{
			ArtifactType: "custom", Version: "v1", SubscriptionName: subscription,
			Metadata: &apiextensionsv1.JSON{Raw: []byte(`{"number":9007199254740993}`)},
		})
	}
	params, err := toUpsertParams(freight, parent, "warehouse")
	require.NoError(t, err)
	row := database.FreightSnapshot{
		Freight: database.Freight{
			ID: params.ID, Name: params.Name, Alias: params.Alias,
			ProjectID: params.ProjectID, WarehouseID: params.WarehouseID,
			CreatedAt: params.CreatedAt, DiscoveredAt: params.DiscoveredAt,
		},
		FreightContents: params.FreightContents,
	}
	testCases := []struct {
		name      string
		change    func(*kargoapi.Freight)
		changeRow func(*database.FreightSnapshot)
		resync    bool
	}{
		{name: "matching artifacts"},
		{
			name: "JSONB formatting matches Kubernetes JSON",
			changeRow: func(snapshot *database.FreightSnapshot) {
				snapshot.Images = slices.Clone(snapshot.Images)
				snapshot.Artifacts = slices.Clone(snapshot.Artifacts)
				snapshot.Images[0].Annotations = []byte(`{ "build": "123" }`)
				snapshot.Artifacts[0].Metadata = []byte(`{ "number": 9007199254740993 }`)
			},
		},
		{
			name: "reordered artifacts update ordinals", resync: true,
			change: func(f *kargoapi.Freight) {
				slices.Reverse(f.Commits)
				slices.Reverse(f.Images)
				slices.Reverse(f.Charts)
				slices.Reverse(f.Artifacts)
			},
		},
		{
			name: "commit metadata differs", resync: true,
			change: func(f *kargoapi.Freight) { f.Commits[0].Message = "changed" },
		},
		{
			name: "image annotations differ", resync: true,
			change: func(f *kargoapi.Freight) { f.Images[0].Annotations["build"] = "456" },
		},
		{
			name: "chart version differs", resync: true,
			change: func(f *kargoapi.Freight) { f.Charts[0].Version = "2.0" },
		},
		{
			name: "artifact metadata differs", resync: true,
			change: func(f *kargoapi.Freight) { f.Artifacts[0].Metadata = nil },
		},
		{
			name: "artifact missing", resync: true,
			change: func(f *kargoapi.Freight) { f.Images = f.Images[:1] },
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			obj := freight.DeepCopy()
			if testCase.change != nil {
				testCase.change(obj)
			}
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			reader := fake.NewClientBuilder().WithScheme(scheme).WithObjects(parent.DeepCopy(), obj).Build()
			snapshot := row
			if testCase.changeRow != nil {
				testCase.changeRow(&snapshot)
			}
			store := &fakeStore{rows: []database.FreightSnapshot{snapshot}}
			changes, diffErr := NewSyncer(reader, store).Diff(context.Background())
			require.NoError(t, diffErr)
			require.Empty(t, changes.ToDelete)
			if testCase.resync {
				require.Equal(t, []client.ObjectKey{{Namespace: "demo", Name: "bundle"}}, changes.ToSync)
			} else {
				require.Empty(t, changes.ToSync)
			}
		})
	}
}
