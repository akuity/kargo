package acr

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestProjectIdentityName(t *testing.T) {
	t.Parallel()
	require.Equal(t, "kargo-project-demo", projectIdentityName("demo"))
}

// fakeIdentityGetter is a userAssignedIdentityGetter that returns canned
// results and counts the calls it receives.
type fakeIdentityGetter struct {
	clientID string
	err      error
	calls    int
}

func (f *fakeIdentityGetter) Get(
	context.Context,
	string,
	string,
	*armmsi.UserAssignedIdentitiesClientGetOptions,
) (armmsi.UserAssignedIdentitiesClientGetResponse, error) {
	f.calls++
	if f.err != nil {
		return armmsi.UserAssignedIdentitiesClientGetResponse{}, f.err
	}
	res := armmsi.UserAssignedIdentitiesClientGetResponse{}
	if f.clientID != "" {
		res.Properties = &armmsi.UserAssignedIdentityProperties{
			ClientID: ptr.To(f.clientID),
		}
	}
	return res, nil
}

// newTestResolvingProvider returns a provider equipped to resolve client IDs
// through the given getter.
func newTestResolvingProvider(
	getter userAssignedIdentityGetter,
) *WorkloadIdentityProvider {
	return &WorkloadIdentityProvider{
		resourceGroup: "test-rg",
		identities:    getter,
	}
}

func TestProjectIdentityResolveClientID(t *testing.T) {
	const testClientID = "11111111-2222-3333-4444-555555555555"

	testCases := []struct {
		name   string
		getter *fakeIdentityGetter
		assert func(t *testing.T, clientID string, err error)
	}{
		{
			name:   "identity found",
			getter: &fakeIdentityGetter{clientID: testClientID},
			assert: func(t *testing.T, clientID string, err error) {
				require.NoError(t, err)
				require.Equal(t, testClientID, clientID)
			},
		},
		{
			name: "identity does not exist",
			getter: &fakeIdentityGetter{
				err: &azcore.ResponseError{StatusCode: http.StatusNotFound},
			},
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorIs(t, err, errNoProjectIdentity)
			},
		},
		{
			name: "caller is not authorized",
			getter: &fakeIdentityGetter{
				err: &azcore.ResponseError{StatusCode: http.StatusForbidden},
			},
			assert: func(t *testing.T, _ string, err error) {
				require.NotErrorIs(t, err, errNoProjectIdentity)
				require.ErrorContains(t, err, "error getting managed identity")
			},
		},
		{
			name:   "identity carries no client ID",
			getter: &fakeIdentityGetter{},
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorIs(t, err, errNoProjectIdentity)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			p := newTestResolvingProvider(testCase.getter)
			clientID, err := p.resolveClientID(context.Background(), "demo")
			testCase.assert(t, clientID, err)
		})
	}

	t.Run("every call reaches Azure", func(t *testing.T) {
		t.Parallel()
		// Nothing is retained between calls, so an identity deleted and
		// recreated between them is seen as it is now, not as it was.
		getter := &fakeIdentityGetter{clientID: testClientID}
		p := newTestResolvingProvider(getter)
		for range 3 {
			clientID, resolveErr := p.resolveClientID(context.Background(), "demo")
			require.NoError(t, resolveErr)
			require.Equal(t, testClientID, clientID)
		}
		require.Equal(t, 3, getter.calls)
	})

	t.Run("unexpected error types are not mistaken for absence", func(t *testing.T) {
		t.Parallel()
		getter := &fakeIdentityGetter{err: errors.New("connection refused")}
		p := newTestResolvingProvider(getter)
		_, err := p.resolveClientID(context.Background(), "demo")
		require.NotErrorIs(t, err, errNoProjectIdentity)
	})
}
