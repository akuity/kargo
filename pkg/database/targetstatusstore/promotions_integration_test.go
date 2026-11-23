//go:build integration && db

package targetstatusstore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database/databasetest"
)

func TestListPromotionsIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	f := newFixture(t, pool, "fleet")
	ctx := context.Background()
	f.stage("fleet-prod", "prod")
	f.stage("fleet-canary", "canary")
	a, b := f.target("a"), f.target("b")

	r := f.rollout("prod", "Succeeded", a, b)
	first := f.promotion("prod", "a", "Succeeded", &r, "v1")
	f.promotion("prod", "b", "Succeeded", &r, "v1")
	second := f.promotion("canary", "a", "Failed", nil, "v2")
	third := f.promotion("prod", "a", "", nil, "v2")
	// Two Promotions created in the same instant are ordered by id.
	f.clock = f.clock.Add(-time.Minute)
	tied := f.promotion("canary", "a", "Running", nil, "v3")

	// Another Project's Target of the same name has a history of its own.
	other := newFixture(t, pool, "other")
	other.stage("other-prod", "prod")
	other.target("a")
	other.promotion("prod", "a", "Succeeded", nil, "v9")

	names := func(promotions []Promotion) []string {
		out := make([]string, len(promotions))
		for i, promotion := range promotions {
			out[i] = promotion.Name
		}
		return out
	}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "a Target's Promotions are listed newest first, from every Stage",
			run: func(t *testing.T) {
				promotions, err := f.store.ListPromotions(ctx, "fleet", "a", PromotionListOptions{})
				require.NoError(t, err)
				require.Equal(t, []string{tied, third, second, first}, names(promotions))

				require.Equal(t, "canary", promotions[0].Stage)
				require.Equal(t, "Running", promotions[0].Phase)
				require.Nil(t, promotions[0].FinishedAt)
				// One the controller has not picked up yet is pending.
				require.Equal(t, "Pending", promotions[1].Phase)
				require.Equal(t, "Failed", promotions[2].Phase)
				require.Equal(t, "v2", promotions[2].Freight)
				require.Zero(t, promotions[2].Rollout)
				require.NotNil(t, promotions[2].FinishedAt)
				require.Equal(t, "prod", promotions[3].Stage)
				require.Equal(t, r.number, promotions[3].Rollout)
				require.True(t, promotions[3].CreatedAt.Before(promotions[2].CreatedAt))
			},
		},
		{
			name: "a Stage narrows the list",
			run: func(t *testing.T) {
				promotions, err := f.store.ListPromotions(
					ctx, "fleet", "a", PromotionListOptions{Stage: "prod"},
				)
				require.NoError(t, err)
				require.Equal(t, []string{third, first}, names(promotions))
			},
		},
		{
			name: "a page continues before the Promotion the last one ended at",
			run: func(t *testing.T) {
				var read []string
				opts := PromotionListOptions{Limit: 1}
				for {
					promotions, err := f.store.ListPromotions(ctx, "fleet", "a", opts)
					require.NoError(t, err)
					if len(promotions) == 0 {
						break
					}
					require.Len(t, promotions, 1)
					read = append(read, promotions[0].Name)
					opts.Before = &PromotionCursor{
						CreatedAt: promotions[0].CreatedAt,
						ID:        promotions[0].ID,
					}
				}
				require.Equal(t, []string{tied, third, second, first}, read)
			},
		},
		{
			name: "a Target with no Promotions has an empty history",
			run: func(t *testing.T) {
				for _, target := range []string{"no-such-target", ""} {
					promotions, err := f.store.ListPromotions(ctx, "fleet", target, PromotionListOptions{})
					require.NoError(t, err)
					require.NotNil(t, promotions)
					require.Empty(t, promotions)
				}
				promotions, err := f.store.ListPromotions(ctx, "no-such-project", "a", PromotionListOptions{})
				require.NoError(t, err)
				require.Empty(t, promotions)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.run(t)
		})
	}
}
