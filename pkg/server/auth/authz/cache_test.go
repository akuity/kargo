package authz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"

	"github.com/akuity/kargo/pkg/server/user"
)

// countingAuthorizer answers every question the same way and counts how often
// it is asked.
type countingAuthorizer struct {
	allowed bool
	err     error
	calls   int
}

func (c *countingAuthorizer) Authorize(
	context.Context,
	user.Subject,
	authv1.ResourceAttributes,
) (bool, error) {
	c.calls++
	return c.allowed, c.err
}

type fakeClock struct{ now time.Time }

func (f *fakeClock) Now() time.Time { return f.now }

func TestWithDecisionCache(t *testing.T) {
	t.Parallel()
	next := &countingAuthorizer{}
	require.Same(t, next, WithDecisionCache(next, 0, 0))
	require.NotSame(t, next, WithDecisionCache(next, time.Minute, 0))
	require.NotSame(t, next, WithDecisionCache(next, 0, time.Minute))
}

func TestDecisionCache_Authorize(t *testing.T) {
	t.Parallel()
	const (
		allowTTL = 5 * time.Minute
		denyTTL  = 30 * time.Second
	)
	start := time.Now()
	subject := user.Subject{Username: "system:serviceaccount:kargo-demo:viewer"}
	getStage := authv1.ResourceAttributes{
		Verb:      "get",
		Group:     "kargo.akuity.io",
		Resource:  "stages",
		Namespace: "kargo-demo",
		Name:      "uat",
	}
	ask := func(t *testing.T, c *decisionCache, s user.Subject, ra authv1.ResourceAttributes) (bool, error) {
		t.Helper()
		return c.Authorize(t.Context(), s, ra)
	}
	testCases := []struct {
		name   string
		next   *countingAuthorizer
		assert func(*testing.T, *countingAuthorizer, *decisionCache, *fakeClock)
	}{
		{
			name: "allow is remembered for the allow TTL",
			next: &countingAuthorizer{allowed: true},
			assert: func(t *testing.T, next *countingAuthorizer, c *decisionCache, clock *fakeClock) {
				for range 3 {
					allowed, err := ask(t, c, subject, getStage)
					require.NoError(t, err)
					require.True(t, allowed)
				}
				require.Equal(t, 1, next.calls)
				clock.now = start.Add(allowTTL - time.Second)
				_, _ = ask(t, c, subject, getStage)
				require.Equal(t, 1, next.calls)
				clock.now = start.Add(allowTTL + time.Second)
				_, _ = ask(t, c, subject, getStage)
				require.Equal(t, 2, next.calls)
			},
		},
		{
			name: "denial is remembered for the deny TTL",
			next: &countingAuthorizer{},
			assert: func(t *testing.T, next *countingAuthorizer, c *decisionCache, clock *fakeClock) {
				allowed, err := ask(t, c, subject, getStage)
				require.NoError(t, err)
				require.False(t, allowed)
				_, _ = ask(t, c, subject, getStage)
				require.Equal(t, 1, next.calls)
				clock.now = start.Add(denyTTL + time.Second)
				_, _ = ask(t, c, subject, getStage)
				require.Equal(t, 2, next.calls)
			},
		},
		{
			name: "failure to answer is not remembered",
			next: &countingAuthorizer{err: errors.New("connection refused")},
			assert: func(t *testing.T, next *countingAuthorizer, c *decisionCache, _ *fakeClock) {
				for range 2 {
					allowed, err := ask(t, c, subject, getStage)
					require.ErrorContains(t, err, "connection refused")
					require.False(t, allowed)
				}
				require.Equal(t, 2, next.calls)
			},
		},
		{
			name: "a different operation is asked on its own",
			next: &countingAuthorizer{allowed: true},
			assert: func(t *testing.T, next *countingAuthorizer, c *decisionCache, _ *fakeClock) {
				_, _ = ask(t, c, subject, getStage)
				promote := getStage
				promote.Verb = "promote"
				_, _ = ask(t, c, subject, promote)
				otherStage := getStage
				otherStage.Name = "prod"
				_, _ = ask(t, c, subject, otherStage)
				require.Equal(t, 3, next.calls)
			},
		},
		{
			// Two users with the same name but different groups are different
			// questions to RBAC.
			name: "a subject differing only in groups is asked on its own",
			next: &countingAuthorizer{allowed: true},
			assert: func(t *testing.T, next *countingAuthorizer, c *decisionCache, _ *fakeClock) {
				alice := user.Subject{Username: "alice", Groups: []string{"viewers"}}
				admin := user.Subject{Username: "alice", Groups: []string{"admins"}}
				_, _ = ask(t, c, alice, getStage)
				_, _ = ask(t, c, admin, getStage)
				require.Equal(t, 2, next.calls)
			},
		},
		{
			// Every OIDC user mapped to the same ServiceAccount is reviewed
			// as the same subject.
			name: "the same subject shares its answers",
			next: &countingAuthorizer{allowed: true},
			assert: func(t *testing.T, next *countingAuthorizer, c *decisionCache, _ *fakeClock) {
				_, _ = ask(t, c, subject, getStage)
				_, _ = ask(t, c, user.Subject{Username: subject.Username}, getStage)
				require.Equal(t, 1, next.calls)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			clock := &fakeClock{now: start}
			c := newDecisionCache(testCase.next, allowTTL, denyTTL, clock)
			testCase.assert(t, testCase.next, c, clock)
		})
	}
}

func TestDecisionCache_disabledKind(t *testing.T) {
	t.Parallel()
	// Only allows are cached; denials are asked every time.
	next := &countingAuthorizer{}
	c := newDecisionCache(next, time.Minute, 0, &fakeClock{now: time.Now()})
	subject := user.Subject{Username: "alice"}
	for range 2 {
		_, _ = c.Authorize(t.Context(), subject, authv1.ResourceAttributes{Verb: "get"})
	}
	require.Equal(t, 2, next.calls)
}
