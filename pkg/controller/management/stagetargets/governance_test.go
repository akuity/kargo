package stagetargets

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
)

func TestGovernance(t *testing.T) {
	t.Parallel()
	us := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	eu := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	bare := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	targets := []database.ListTargetLabelsRow{
		{ID: us, Name: "us", Labels: map[string]string{"region": "us", "tier": "canary"}},
		{ID: eu, Name: "eu", Labels: map[string]string{"region": "eu"}},
		{ID: bare, Name: "bare"},
	}
	stage := func(id, selectors string) database.ListStageSelectorsRow {
		return database.ListStageSelectorsRow{ID: id, Name: id, TargetSelectors: json.RawMessage(selectors)}
	}
	testCases := []struct {
		name   string
		stages []database.ListStageSelectorsRow
		want   []pair
	}{
		{name: "no Stages govern nothing"},
		{name: "an empty selector list governs nothing", stages: []database.ListStageSelectorsRow{stage("s", `[]`)}},
		{
			name:   "an empty selector governs every Target",
			stages: []database.ListStageSelectorsRow{stage("s", `[{}]`)},
			want:   []pair{{"s", us}, {"s", eu}, {"s", bare}},
		},
		{
			name: "a Target matching any one selector is governed",
			stages: []database.ListStageSelectorsRow{stage("s",
				`[{"matchLabels":{"region":"eu"}},{"matchExpressions":[{"key":"tier","operator":"In","values":["canary"]}]}]`,
			)},
			want: []pair{{"s", us}, {"s", eu}},
		},
		{
			name: "NotIn matches a Target without the key, as Kubernetes does",
			stages: []database.ListStageSelectorsRow{stage("s",
				`[{"matchExpressions":[{"key":"region","operator":"NotIn","values":["us"]}]}]`,
			)},
			want: []pair{{"s", eu}, {"s", bare}},
		},
		{
			name: "a Stage whose selectors do not parse governs nothing, and the rest still do",
			stages: []database.ListStageSelectorsRow{
				stage("bad-operator", `[{"matchExpressions":[{"key":"region","operator":"Bogus"}]}]`),
				stage("not-selectors", `[1]`),
				stage("good", `[{"matchLabels":{"region":"us"}}]`),
			},
			want: []pair{{"good", us}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := governance(context.Background(), testCase.stages, targets)
			want := map[pair]struct{}{}
			for _, p := range testCase.want {
				want[p] = struct{}{}
			}
			require.Equal(t, want, got)
		})
	}
}
