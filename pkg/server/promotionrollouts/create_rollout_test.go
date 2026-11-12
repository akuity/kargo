package promotionrollouts

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/user"
)

// TestCreator_Create exercises the Creator directly, as the promote
// endpoints call it, without the create endpoint's validation in front of
// it.
func TestCreator_Create(t *testing.T) {
	t.Parallel()

	// The Stage governs Targets in region us: two of the Project's three.
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "prod"},
		Spec: kargoapi.StageSpec{
			Targets: &kargoapi.StageTargets{
				Selectors: []metav1.LabelSelector{{MatchLabels: map[string]string{"region": "us"}}},
			},
		},
	}
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "fake-freight"},
	}
	fleet := fakeTargets{rows: []database.TargetRow{
		targetRow("us-west", map[string]string{"region": "us"}),
		targetRow("eu-west", map[string]string{"region": "eu"}),
		targetRow("us-east", map[string]string{"region": "us"}),
	}}
	alice := user.KubernetesUser{UserInfo: authnv1.UserInfo{Username: "alice"}}

	testCases := []struct {
		name        string
		store       *fakeStore
		identity    user.Identity
		targetNames []string
		assert      func(*testing.T, *fakeStore, *kargoapi.PromotionRollout, error)
	}{
		{
			name:  "no identity records no creator",
			store: &fakeStore{targets: fleet.rows},
			assert: func(t *testing.T, store *fakeStore, rollout *kargoapi.PromotionRollout, err error) {
				require.NoError(t, err)
				require.NotContains(t, rollout.Annotations, kargoapi.AnnotationKeyCreateActor)
				stored, err := store.Get(t.Context(), testProject, rollout.Name)
				require.NoError(t, err)
				require.Empty(t, stored.CreatedBy)
			},
		},
		{
			name:     "the identity is recorded as the creator",
			store:    &fakeStore{targets: fleet.rows},
			identity: alice,
			assert: func(t *testing.T, store *fakeStore, rollout *kargoapi.PromotionRollout, err error) {
				require.NoError(t, err)
				require.Equal(t, alice.Actor(), rollout.Annotations[kargoapi.AnnotationKeyCreateActor])
				stored, err := store.Get(t.Context(), testProject, rollout.Name)
				require.NoError(t, err)
				require.Equal(t, alice.Actor(), stored.CreatedBy)
			},
		},
		{
			name:        "a name given twice counts once",
			store:       &fakeStore{targets: fleet.rows},
			targetNames: []string{"us-east", "us-east"},
			assert: func(t *testing.T, _ *fakeStore, rollout *kargoapi.PromotionRollout, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.PromotionRolloutTarget{{Name: "us-east"}}, rollout.Spec.Targets)
			},
		},
		{
			name:        "a name the Stage does not govern is a bad request",
			store:       &fakeStore{targets: fleet.rows},
			targetNames: []string{"us-east", "eu-west"},
			assert: func(t *testing.T, store *fakeStore, rollout *kargoapi.PromotionRollout, err error) {
				require.Nil(t, rollout)
				var httpErr *libhttp.HTTPError
				require.ErrorAs(t, err, &httpErr)
				require.Equal(t, http.StatusBadRequest, httpErr.Code())
				require.ErrorContains(t, err, "eu-west")
				require.Zero(t, store.calls)
			},
		},
		{
			name:  "a store error is reported as the API reports it",
			store: &fakeStore{err: fmt.Errorf("Target gone: %w", database.ErrConflict)},
			assert: func(t *testing.T, _ *fakeStore, rollout *kargoapi.PromotionRollout, err error) {
				require.Nil(t, rollout)
				require.True(t, apierrors.IsConflict(err))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			if testCase.identity != nil {
				ctx = user.ContextWithIdentity(ctx, testCase.identity)
			}
			rollout, err := NewCreator(testCase.store, fleet).Create(
				ctx,
				stage,
				freight,
				testCase.targetNames,
				nil,
			)
			testCase.assert(t, testCase.store, rollout, err)
		})
	}
}
