package freight

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/namer"
)

func TestGetAvailableFreightAlias(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	notMayFourth := func() time.Time {
		return time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC)
	}
	mayFourth := func() time.Time {
		return time.Date(2026, time.May, 4, 12, 0, 0, 0, time.UTC)
	}

	testCases := []struct {
		name    string
		webhook *webhook
		assert  func(*testing.T, string, error)
	}{
		{
			name: "error listing Freight",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(scheme).
					WithInterceptorFuncs(interceptor.Funcs{
						List: func(
							context.Context,
							client.WithWatch,
							client.ObjectList,
							...client.ListOption,
						) error {
							return errors.New("something went wrong")
						},
					}).Build(),
				freightAliasGenerator: &fakeNamer{names: []string{"a-b"}},
				nowFn:                 notMayFourth,
			},
			assert: func(t *testing.T, alias string, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(
					t, err, `error checking for existence of Freight with alias "a-b"`,
				)
				require.Empty(t, alias)
			},
		},
		{
			name: "default nouns on an ordinary day",
			webhook: &webhook{
				client:                  fake.NewClientBuilder().WithScheme(scheme).Build(),
				freightAliasGenerator:   &fakeNamer{names: []string{"mortal-dragonfly"}},
				mayFourthAliasGenerator: &fakeNamer{names: []string{"mortal-wookiee"}},
				nowFn:                   notMayFourth,
			},
			assert: func(t *testing.T, alias string, err error) {
				require.NoError(t, err)
				require.Equal(t, "mortal-dragonfly", alias)
			},
		},
		{
			name: "May Fourth nouns on May 4",
			webhook: &webhook{
				client:                  fake.NewClientBuilder().WithScheme(scheme).Build(),
				freightAliasGenerator:   &fakeNamer{names: []string{"mortal-dragonfly"}},
				mayFourthAliasGenerator: &fakeNamer{names: []string{"mortal-wookiee"}},
				nowFn:                   mayFourth,
			},
			assert: func(t *testing.T, alias string, err error) {
				require.NoError(t, err)
				require.Equal(t, "mortal-wookiee", alias)
			},
		},
		{
			name: "retries on collision",
			webhook: &webhook{
				client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-freight",
							Labels:    map[string]string{kargoapi.LabelKeyAlias: "taken-alias"},
						},
					},
				).Build(),
				freightAliasGenerator: &fakeNamer{
					names: []string{"taken-alias", "free-alias"},
				},
				nowFn: notMayFourth,
			},
			assert: func(t *testing.T, alias string, err error) {
				require.NoError(t, err)
				require.Equal(t, "free-alias", alias)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			alias, err := testCase.webhook.getAvailableFreightAlias(
				context.Background(),
			)
			testCase.assert(t, alias, err)
		})
	}
}

// TestMayFourthNouns guards against typos that would produce aliases unfit for
// use as label values.
func TestMayFourthNouns(t *testing.T) {
	t.Parallel()
	_, err := namer.New(namer.DefaultDescriptors, mayFourthNouns)
	require.NoError(t, err)
	wordRegex := regexp.MustCompile(`^[a-z0-9]+$`)
	seen := make(map[string]struct{}, len(mayFourthNouns))
	for _, noun := range mayFourthNouns {
		require.Regexp(t, wordRegex, noun)
		require.NotContains(t, seen, noun, "duplicate noun %q", noun)
		seen[noun] = struct{}{}
	}
}

// fakeNamer returns its names in order, repeating the last one indefinitely.
type fakeNamer struct {
	names []string
	i     int
}

func (f *fakeNamer) Name() string {
	name := f.names[f.i]
	if f.i < len(f.names)-1 {
		f.i++
	}
	return name
}
