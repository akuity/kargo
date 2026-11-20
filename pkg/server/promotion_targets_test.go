package server

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/promotionrollouts"
)

// rolloutCreation is what a promote endpoint asked the fake creator for.
type rolloutCreation struct {
	stage          string
	freight        string
	targets        []string
	updateStrategy *kargoapi.TargetUpdateStrategy
}

// fakeRolloutCreator is a promotionrollouts.Creator that records what it is
// asked to create and answers with a rollout of the Stage's name, or with
// the error configured for the Stage.
type fakeRolloutCreator struct {
	mu        sync.Mutex
	creations []rolloutCreation
	// errs fails the creation of a rollout for the named Stage.
	errs map[string]error
}

var _ promotionrollouts.Creator = (*fakeRolloutCreator)(nil)

func (f *fakeRolloutCreator) Create(
	_ context.Context,
	stage *kargoapi.Stage,
	freight *kargoapi.Freight,
	targetNames []string,
	updateStrategy *kargoapi.TargetUpdateStrategy,
) (*kargoapi.PromotionRollout, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creations = append(f.creations, rolloutCreation{
		stage:          stage.Name,
		freight:        freight.Name,
		targets:        targetNames,
		updateStrategy: updateStrategy,
	})
	if err := f.errs[stage.Name]; err != nil {
		return nil, err
	}
	targets := make([]kargoapi.PromotionRolloutTarget, len(targetNames))
	for i, name := range targetNames {
		targets[i] = kargoapi.PromotionRolloutTarget{Name: name}
	}
	return &kargoapi.PromotionRollout{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: stage.Namespace,
			Name:      stage.Name + ".fake-rollout",
		},
		Number: int64(len(f.creations)),
		Spec: kargoapi.PromotionRolloutSpec{
			Stage:   stage.Name,
			Freight: freight.Name,
			Targets: targets,
		},
		Status: kargoapi.PromotionRolloutStatus{Phase: kargoapi.PromotionRolloutPhasePending},
	}, nil
}

func Test_server_createPromotionRollout(t *testing.T) {
	t.Parallel()
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project", Name: "fake-stage"},
		Spec: kargoapi.StageSpec{
			Targets: &kargoapi.StageTargets{
				Selectors: []metav1.LabelSelector{{MatchLabels: map[string]string{"region": "us"}}},
			},
		},
	}
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project", Name: "fake-freight"},
	}
	strategy := &kargoapi.TargetUpdateStrategy{MaxConcurrent: new(intstr.FromInt32(1))}
	errRefused := libhttp.ErrorStr("refused", http.StatusBadRequest)

	testCases := []struct {
		name    string
		creator *fakeRolloutCreator
		assert  func(*testing.T, *fakeRolloutCreator, *kargoapi.PromotionRollout, error)
	}{
		{
			name: "without a database",
			assert: func(t *testing.T, _ *fakeRolloutCreator, rollout *kargoapi.PromotionRollout, err error) {
				require.ErrorIs(t, err, errDatabaseNotConfigured)
				require.Nil(t, rollout)
			},
		},
		{
			name:    "the creator's error is passed on",
			creator: &fakeRolloutCreator{errs: map[string]error{"fake-stage": errRefused}},
			assert: func(t *testing.T, _ *fakeRolloutCreator, rollout *kargoapi.PromotionRollout, err error) {
				require.ErrorIs(t, err, errRefused)
				require.Nil(t, rollout)
			},
		},
		{
			name:    "the Targets and pace asked for are passed on",
			creator: &fakeRolloutCreator{},
			assert: func(t *testing.T, creator *fakeRolloutCreator, rollout *kargoapi.PromotionRollout, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-stage", rollout.Spec.Stage)
				require.Equal(t, "fake-freight", rollout.Spec.Freight)
				require.Equal(t, []rolloutCreation{{
					stage:          "fake-stage",
					freight:        "fake-freight",
					targets:        []string{"us-east"},
					updateStrategy: strategy,
				}}, creator.creations)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			s := &server{}
			if testCase.creator != nil {
				s.rolloutCreator = testCase.creator
			}
			rollout, err := s.createPromotionRollout(
				t.Context(), stage, freight, []string{"us-east"}, strategy,
			)
			testCase.assert(t, testCase.creator, rollout, err)
		})
	}
}

func Test_server_getListTargetsFn(t *testing.T) {
	t.Parallel()
	t.Run("without a database", func(t *testing.T) {
		t.Parallel()
		_, err := (&server{}).getListTargetsFn()
		require.ErrorIs(t, err, errDatabaseNotConfigured)
	})
	t.Run("with a database", func(t *testing.T) {
		t.Parallel()
		errListing := errors.New("listing failed")
		s := &server{
			listTargetsFn: func(context.Context, string) ([]kargoapi.Target, error) {
				return nil, errListing
			},
		}
		fn, err := s.getListTargetsFn()
		require.NoError(t, err)
		_, err = fn(t.Context(), "fake-project")
		require.ErrorIs(t, err, errListing)
	})
}
