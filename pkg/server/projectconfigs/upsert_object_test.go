package projectconfigs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

func TestUpsertObject(t *testing.T) {
	t.Parallel()
	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
		Name: testProject,
		UID:  testProjectUID,
	}}
	validObj := func() map[string]any {
		return map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       kind,
			"metadata": map[string]any{
				"name":      testProject,
				"namespace": testProject,
			},
			"spec": map[string]any{
				"freightLinks": []any{
					map[string]any{"title": "t", "url": "https://example.com"},
				},
			},
		}
	}
	existingStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(kargoapi.ProjectConfigSpec{})
		return store
	}

	testCases := []struct {
		name      string
		store     *fakeStore
		authorize func(context.Context, can.Access) error
		obj       map[string]any
		opts      UpsertOptions
		assert    func(*testing.T, *fakeStore, *kargoapi.ProjectConfig, bool, error)
	}{
		{
			name:      "not authorized",
			authorize: forbidEverything,
			obj:       validObj(),
			assert: func(t *testing.T, store *fakeStore, _ *kargoapi.ProjectConfig, _ bool, err error) {
				require.True(t, apierrors.IsForbidden(err))
				require.Zero(t, store.calls)
			},
		},
		{
			name: "decode errors are reported as for PUT",
			obj: map[string]any{
				"metadata": map[string]any{"name": testProject},
				"spec": map[string]any{
					"freightLinks": []any{map[string]any{"title": "t"}},
				},
			},
			assert: func(t *testing.T, store *fakeStore, _ *kargoapi.ProjectConfig, _ bool, err error) {
				require.True(t, apierrors.IsInvalid(err))
				require.ErrorContains(t, err, "spec.freightLinks[0].url")
				require.Empty(t, store.configs)
			},
		},
		{
			name: "creates when absent",
			obj:  validObj(),
			assert: func(t *testing.T, store *fakeStore, out *kargoapi.ProjectConfig, created bool, err error) {
				require.NoError(t, err)
				require.True(t, created)
				require.Equal(t, kind, out.Kind)
				require.Len(t, out.Spec.FreightLinks, 1)
				require.Equal(t, testProjectUID, store.configs[testProject].ProjectUID)
			},
		},
		{
			name:  "updates when present",
			store: existingStore(),
			obj:   validObj(),
			assert: func(t *testing.T, _ *fakeStore, out *kargoapi.ProjectConfig, created bool, err error) {
				require.NoError(t, err)
				require.False(t, created)
				require.Equal(t, int64(2), out.Generation)
			},
		},
		{
			name:      "create only refuses an existing ProjectConfig",
			store:     existingStore(),
			authorize: forbid(can.Update()),
			obj:       validObj(),
			opts:      UpsertOptions{CreateOnly: true},
			assert: func(t *testing.T, store *fakeStore, _ *kargoapi.ProjectConfig, _ bool, err error) {
				require.True(t, apierrors.IsAlreadyExists(err))
				require.EqualError(
					t,
					err,
					`projectconfigs.kargo.akuity.io "`+testProject+`" already exists`,
				)
				require.Nil(t, store.written)
			},
		},
		{
			name:  "create only refuses a ProjectConfig another request created first",
			store: &fakeStore{raceOnCreate: true},
			obj:   validObj(),
			opts:  UpsertOptions{CreateOnly: true},
			assert: func(t *testing.T, store *fakeStore, _ *kargoapi.ProjectConfig, _ bool, err error) {
				require.True(t, apierrors.IsAlreadyExists(err))
				require.Equal(t, int64(1), store.configs[testProject].Generation)
			},
		},
		{
			name: "create only creates when absent",
			obj:  validObj(),
			opts: UpsertOptions{CreateOnly: true},
			assert: func(t *testing.T, store *fakeStore, _ *kargoapi.ProjectConfig, created bool, err error) {
				require.NoError(t, err)
				require.True(t, created)
				require.Contains(t, store.configs, testProject)
			},
		},
		{
			name:      "must exist refuses to create",
			authorize: forbid(can.Create()),
			obj:       validObj(),
			opts:      UpsertOptions{MustExist: true},
			assert: func(t *testing.T, store *fakeStore, _ *kargoapi.ProjectConfig, _ bool, err error) {
				require.True(t, apierrors.IsNotFound(err))
				require.Nil(t, store.written)
				require.Empty(t, store.configs)
			},
		},
		{
			name: "must exist ignores a leftover of an earlier Project",
			store: func() *fakeStore {
				return leftoverStore()
			}(),
			obj:  validObj(),
			opts: UpsertOptions{MustExist: true},
			assert: func(t *testing.T, store *fakeStore, _ *kargoapi.ProjectConfig, _ bool, err error) {
				require.True(t, apierrors.IsNotFound(err))
				require.Nil(t, store.written)
			},
		},
		{
			name:  "must exist updates when present",
			store: existingStore(),
			obj:   validObj(),
			opts:  UpsertOptions{MustExist: true},
			assert: func(t *testing.T, _ *fakeStore, _ *kargoapi.ProjectConfig, created bool, err error) {
				require.NoError(t, err)
				require.False(t, created)
			},
		},
		{
			name:      "skipping user authorization creates without asking",
			authorize: forbidEverything,
			obj:       validObj(),
			opts:      UpsertOptions{SkipUserAuthorization: true},
			assert: func(t *testing.T, store *fakeStore, _ *kargoapi.ProjectConfig, created bool, err error) {
				require.NoError(t, err)
				require.True(t, created)
				require.Contains(t, store.configs, testProject)
			},
		},
		{
			name:      "skipping user authorization updates without asking",
			store:     existingStore(),
			authorize: forbidEverything,
			obj:       validObj(),
			opts:      UpsertOptions{SkipUserAuthorization: true},
			assert: func(t *testing.T, _ *fakeStore, _ *kargoapi.ProjectConfig, created bool, err error) {
				require.NoError(t, err)
				require.False(t, created)
			},
		},
		{
			name:      "skipping user authorization still validates",
			authorize: forbidEverything,
			obj: map[string]any{
				"metadata": map[string]any{"name": "another-project"},
			},
			opts: UpsertOptions{SkipUserAuthorization: true},
			assert: func(t *testing.T, store *fakeStore, _ *kargoapi.ProjectConfig, _ bool, err error) {
				require.True(t, apierrors.IsInvalid(err))
				require.Empty(t, store.configs)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := tc.store
			if store == nil {
				store = &fakeStore{}
			}
			authorize := tc.authorize
			if authorize == nil {
				authorize = func(context.Context, can.Access) error { return nil }
			}
			out, created, err := New(store, authorize, nil).UpsertObject(
				t.Context(),
				project,
				tc.obj,
				tc.opts,
			)
			tc.assert(t, store, out, created, err)
		})
	}
}

func TestUpsertRejectsNilProjectConfig(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	_, _, err := New(
		store,
		func(context.Context, can.Access) error { return nil },
		nil,
	).Upsert(
		t.Context(),
		&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: testProject}},
		nil,
	)
	require.Error(t, err)
	require.Zero(t, store.calls)
}
