package garbage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCleanProjects(t *testing.T) {
	testCases := []struct {
		name         string
		collector    *collector
		errHandlerFn func(ctx context.Context, errCh <-chan struct{})
	}{
		{
			// The objective of this test case is to ensure that errCh is signaled
			// when an error occurs.
			name: "error cleaning individual Project",
			collector: &collector{
				cleanProjectFn: func(context.Context, string) error {
					return errors.New("something went wrong")
				},
			},
			errHandlerFn: func(ctx context.Context, errCh <-chan struct{}) {
				select {
				case _, ok := <-errCh:
					if !ok {
						require.FailNow(
							t,
							"error channel was closed without receiving any signals",
						)
					}
				case <-ctx.Done():
					require.FailNow(
						t,
						"timed out without receiving an error signal",
					)
				}
			},
		},

		{
			// The objective of this test case is to ensure that errCh is NOT signaled
			// when everything goes smoothly.
			name: "success",
			collector: &collector{
				cleanProjectFn: func(context.Context, string) error {
					return nil
				},
			},
			errHandlerFn: func(ctx context.Context, errCh <-chan struct{}) {
				select {
				case _, ok := <-errCh:
					if ok {
						require.FailNow(t, "an unexpected error signal was received")
					}
				case <-ctx.Done():
				}
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			projectCh := make(chan string)
			errCh := make(chan struct{})

			go testCase.collector.cleanProjects(ctx, projectCh, errCh)

			select {
			case projectCh <- "fake-project":
			case <-ctx.Done():
				require.FailNow(t, "timed out sending a Project name")
			}

			testCase.errHandlerFn(ctx, errCh)
		})
	}
}

func TestCleanProject(t *testing.T) {
	testCases := []struct {
		name       string
		collector  *collector
		assertions func(*testing.T, error)
	}{
		{
			name: "errors cleaning Promotions and Freight",
			collector: &collector{
				cleanProjectPromotionsFn: func(context.Context, string) error {
					return errors.New("something went wrong")
				},
				cleanProjectFreightFn: func(context.Context, string) error {
					return errors.New("something else went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error cleaning Promotions in Project")
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error cleaning Freight in Project")
				require.ErrorContains(t, err, "something else went wrong")
			},
		},
		{
			name: "success",
			collector: &collector{
				cleanProjectPromotionsFn: func(context.Context, string) error {
					return nil
				},
				cleanProjectFreightFn: func(context.Context, string) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.collector.cleanProject(context.Background(), "fake-project"),
			)
		})
	}
}
