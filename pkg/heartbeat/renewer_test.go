package heartbeat

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewRenewer(t *testing.T) {
	testCases := []struct {
		name           string
		controllerName string
		assertions     func(*testing.T, *renewer)
	}{
		{
			name:           "named controller",
			controllerName: "alpha",
			assertions: func(t *testing.T, r *renewer) {
				require.Equal(t, "alpha", r.controllerName)
				require.Equal(t, "kargo-controller-alpha", r.leaseName)
				require.Equal(t, "kargo", r.namespace)
				require.Equal(t, defaultLeaseDuration, r.leaseDuration)
				require.Equal(t, defaultRenewInterval, r.renewInterval)
				require.NotEmpty(t, r.holderIdentity)
			},
		},
		{
			name:           "unnamed controller",
			controllerName: "",
			assertions: func(t *testing.T, r *renewer) {
				require.Empty(t, r.controllerName,
					"label value stays empty for an unnamed controller")
				require.Equal(t, "kargo-controller-unnamed", r.leaseName,
					"lease name uses 'unnamed' for K8s resource-name uniqueness")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			//nolint:forcetypeassert
			r := NewRenewer(newFakeClient(t), "kargo", testCase.controllerName).(*renewer)
			testCase.assertions(t, r)
		})
	}
}

func TestRenewer_Start(t *testing.T) {
	r, c := newTestRenewer(t, "alpha")
	r.renewInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- r.Start(ctx) }()

	require.Eventually(t, func() bool {
		got := &coordinationv1.Lease{}
		return c.Get(t.Context(), types.NamespacedName{
			Namespace: "kargo",
			Name:      "kargo-controller-alpha",
		}, got) == nil
	}, time.Second, 10*time.Millisecond,
		"renewer should create the lease on Start")

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancel")
	}

	got := &coordinationv1.Lease{}
	err := c.Get(t.Context(), types.NamespacedName{
		Namespace: "kargo",
		Name:      "kargo-controller-alpha",
	}, got)
	require.True(t, apierrors.IsNotFound(err),
		"renewer should delete its lease on shutdown so observers see dead immediately")
}

func TestRenewer_NeedLeaderElection(t *testing.T) {
	r, _ := newTestRenewer(t, "alpha")
	require.False(t, r.NeedLeaderElection(),
		"the renewer must run on every replica, not just a leader")
}

func TestRenewer_renew(t *testing.T) {
	old := time.Now().Add(-1 * time.Hour)

	testCases := []struct {
		name           string
		controllerName string
		objects        []client.Object
		assertions     func(*testing.T, client.Client)
	}{
		{
			name:           "creates lease when absent",
			controllerName: "alpha",
			assertions: func(t *testing.T, c client.Client) {
				got := &coordinationv1.Lease{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: "kargo",
					Name:      "kargo-controller-alpha",
				}, got))
				require.Equal(t, "alpha", got.Labels[kargoapi.LabelKeyController])
				require.NotNil(t, got.Spec.HolderIdentity)
				require.Equal(t, "test-holder", *got.Spec.HolderIdentity)
				require.NotNil(t, got.Spec.LeaseDurationSeconds)
				require.Equal(t,
					int32(defaultLeaseDuration.Seconds()), *got.Spec.LeaseDurationSeconds)
				require.NotNil(t, got.Spec.RenewTime)
				require.NotNil(t, got.Spec.AcquireTime)
			},
		},
		{
			name:           "updates existing lease and preserves AcquireTime",
			controllerName: "alpha",
			objects: []client.Object{
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo",
						Name:      "kargo-controller-alpha",
						Labels:    map[string]string{kargoapi.LabelKeyController: "alpha"},
					},
					Spec: coordinationv1.LeaseSpec{
						RenewTime:   &metav1.MicroTime{Time: old},
						AcquireTime: &metav1.MicroTime{Time: old},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client) {
				got := &coordinationv1.Lease{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: "kargo",
					Name:      "kargo-controller-alpha",
				}, got))
				require.NotNil(t, got.Spec.RenewTime)
				require.True(t, got.Spec.RenewTime.After(old),
					"RenewTime should advance")
				require.NotNil(t, got.Spec.AcquireTime)
				require.True(t,
					got.Spec.AcquireTime.Equal(&metav1.MicroTime{Time: old}),
					"AcquireTime should be preserved across renewals")
			},
		},
		{
			name:           "unnamed controller writes empty label value",
			controllerName: "",
			assertions: func(t *testing.T, c client.Client) {
				got := &coordinationv1.Lease{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: "kargo",
					Name:      "kargo-controller-unnamed",
				}, got))
				value, present := got.Labels[kargoapi.LabelKeyController]
				require.True(t, present, "controller label key must be set")
				require.Empty(t, value,
					"controller label value must be empty for an unnamed controller")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r, c := newTestRenewer(t, testCase.controllerName, testCase.objects...)
			require.NoError(t, r.renew(t.Context()))
			testCase.assertions(t, c)
		})
	}
}

func TestRenewer_delete(t *testing.T) {
	testCases := []struct {
		name       string
		objects    []client.Object
		assertions func(*testing.T, client.Client, error)
	}{
		{
			name: "removes lease",
			objects: []client.Object{
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo",
						Name:      "kargo-controller-alpha",
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				got := &coordinationv1.Lease{}
				lookupErr := c.Get(t.Context(), types.NamespacedName{
					Namespace: "kargo",
					Name:      "kargo-controller-alpha",
				}, got)
				require.True(t, apierrors.IsNotFound(lookupErr))
			},
		},
		{
			name: "is idempotent when lease is already absent",
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.NoError(t, err,
					"deleting an absent lease must not error")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r, c := newTestRenewer(t, "alpha", testCase.objects...)
			testCase.assertions(t, c, r.delete(t.Context()))
		})
	}
}

// newTestRenewer returns a *renewer wired against a fake client preloaded
// with the given objects, ready to drive renew/delete/Start.
func newTestRenewer(
	t *testing.T,
	controllerName string,
	objs ...client.Object,
) (*renewer, client.Client) {
	t.Helper()
	c := newFakeClient(t, objs...)
	r := NewRenewer(c, "kargo", controllerName).(*renewer) //nolint:forcetypeassert
	r.holderIdentity = "test-holder"
	return r, c
}
