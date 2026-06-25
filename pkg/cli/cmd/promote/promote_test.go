package promote

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/watch"
)

func TestPromotionOptionsValidate(t *testing.T) {
	testCases := []struct {
		name       string
		opts       promotionOptions
		assertions func(*testing.T, promotionOptions, error)
	}{
		{
			name: "missing project",
			opts: promotionOptions{
				FreightName: "fake-freight",
				Stage:       "fake-stage",
			},
			assertions: func(t *testing.T, _ promotionOptions, err error) {
				require.ErrorContains(t, err, "project is required")
			},
		},
		{
			name: "origin requires stage",
			opts: promotionOptions{
				Project:        "fake-project",
				Origin:         "Warehouse/fake-warehouse",
				DownstreamFrom: "fake-stage",
			},
			assertions: func(t *testing.T, _ promotionOptions, err error) {
				require.ErrorContains(t, err, "origin can only be used with stage")
			},
		},
		{
			name: "origin with stage",
			opts: promotionOptions{
				Project: "fake-project",
				Origin:  "Warehouse/fake-warehouse",
				Stage:   "fake-stage",
			},
			assertions: func(t *testing.T, _ promotionOptions, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.opts.validate()
			testCase.assertions(t, testCase.opts, err)
		})
	}
}

func TestWaitForTerminalPromotion(t *testing.T) {
	testErr := errors.New("test error")
	testCases := []struct {
		name       string
		setup      func() (<-chan watch.Event[*kargoapi.Promotion], <-chan error)
		assertions func(*testing.T, *kargoapi.Promotion, error)
	}{
		{
			name: "returns terminal promotion",
			setup: func() (<-chan watch.Event[*kargoapi.Promotion], <-chan error) {
				eventCh := make(chan watch.Event[*kargoapi.Promotion], 2)
				errCh := make(chan error, 1)
				eventCh <- watch.Event[*kargoapi.Promotion]{
					Object: &kargoapi.Promotion{
						Status: kargoapi.PromotionStatus{
							Phase: kargoapi.PromotionPhaseRunning,
						},
					},
				}
				eventCh <- watch.Event[*kargoapi.Promotion]{
					Object: &kargoapi.Promotion{
						Status: kargoapi.PromotionStatus{
							Phase: kargoapi.PromotionPhaseSucceeded,
						},
					},
				}
				return eventCh, errCh
			},
			assertions: func(t *testing.T, p *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.Equal(t, kargoapi.PromotionPhaseSucceeded, p.Status.Phase)
			},
		},
		{
			name: "returns watch error",
			setup: func() (<-chan watch.Event[*kargoapi.Promotion], <-chan error) {
				eventCh := make(chan watch.Event[*kargoapi.Promotion])
				close(eventCh)
				errCh := make(chan error, 1)
				errCh <- testErr
				return eventCh, errCh
			},
			assertions: func(t *testing.T, p *kargoapi.Promotion, err error) {
				require.Nil(t, p)
				require.ErrorIs(t, err, testErr)
			},
		},
		{
			name: "returns unexpected end of stream",
			setup: func() (<-chan watch.Event[*kargoapi.Promotion], <-chan error) {
				eventCh := make(chan watch.Event[*kargoapi.Promotion])
				close(eventCh)
				errCh := make(chan error)
				return eventCh, errCh
			},
			assertions: func(t *testing.T, p *kargoapi.Promotion, err error) {
				require.Nil(t, p)
				require.ErrorContains(t, err, "unexpected end of watch stream")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			eventCh, errCh := testCase.setup()
			p, err := waitForTerminalPromotion(context.Background(), eventCh, errCh)
			testCase.assertions(t, p, err)
		})
	}
}
