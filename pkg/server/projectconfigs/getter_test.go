package projectconfigs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

var _ api.ProjectConfigGetter = StoreGetter(nil, nil)

func TestStoreGetter(t *testing.T) {
	t.Parallel()
	project := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: testProject, UID: types.UID(testProjectUID)},
	}
	withConfig := func() *fakeStore {
		s := &fakeStore{}
		s.add(kargoapi.ProjectConfigSpec{})
		return s
	}
	testCases := []struct {
		name      string
		noProject bool
		store     func() *fakeStore
		assert    func(*testing.T, *kargoapi.ProjectConfig, error)
	}{
		{
			name:      "no Project",
			noProject: true,
			store:     withConfig,
			assert: func(t *testing.T, cfg *kargoapi.ProjectConfig, err error) {
				require.NoError(t, err)
				require.Nil(t, cfg)
			},
		},
		{
			name:  "no ProjectConfig",
			store: func() *fakeStore { return &fakeStore{} },
			assert: func(t *testing.T, cfg *kargoapi.ProjectConfig, err error) {
				require.NoError(t, err)
				require.Nil(t, cfg)
			},
		},
		{
			name:  "leftover row of a deleted Project",
			store: leftoverStore,
			assert: func(t *testing.T, cfg *kargoapi.ProjectConfig, err error) {
				require.NoError(t, err)
				require.Nil(t, cfg)
			},
		},
		{
			name:  "store error",
			store: func() *fakeStore { return &fakeStore{err: errors.New("db down")} },
			assert: func(t *testing.T, cfg *kargoapi.ProjectConfig, err error) {
				require.ErrorContains(t, err, "db down")
				require.Nil(t, cfg)
			},
		},
		{
			name:  "found",
			store: withConfig,
			assert: func(t *testing.T, cfg *kargoapi.ProjectConfig, err error) {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				require.Equal(t, testProject, cfg.Name)
				require.Equal(t, testProject, cfg.Namespace)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			builder := fake.NewClientBuilder().WithScheme(scheme)
			if !tc.noProject {
				builder = builder.WithObjects(project.DeepCopy())
			}
			cfg, err := StoreGetter(tc.store(), builder.Build()).Get(t.Context(), testProject)
			tc.assert(t, cfg, err)
		})
	}
}
