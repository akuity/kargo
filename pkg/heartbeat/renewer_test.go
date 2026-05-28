package heartbeat

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewRenewer(t *testing.T) {
	const testNamespace = "kargo"

	testCases := []struct {
		name           string
		controllerName string
		leaseDuration  time.Duration
		assertions     func(*testing.T, *renewer)
	}{
		{
			name:           "named controller",
			controllerName: "alpha",
			leaseDuration:  30 * time.Second,
			assertions: func(t *testing.T, r *renewer) {
				require.Equal(t, "alpha", r.controllerName)
				require.Equal(t, "kargo-controller-alpha", r.leaseName)
				require.Equal(t, testNamespace, r.namespace)
				require.Equal(t, 30*time.Second, r.leaseDuration)
				require.Equal(t, 10*time.Second, r.renewInterval)
				require.NotEmpty(t, r.holderIdentity)
			},
		},
		{
			name:           "unnamed controller",
			controllerName: "",
			leaseDuration:  30 * time.Second,
			assertions: func(t *testing.T, r *renewer) {
				require.Empty(t, r.controllerName)
				require.Equal(t, "kargo-controller-unnamed", r.leaseName)
				require.Equal(t, testNamespace, r.namespace)
				require.Equal(t, 30*time.Second, r.leaseDuration)
				require.Equal(t, 10*time.Second, r.renewInterval)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			//nolint:forcetypeassert
			r := NewRenewer(
				fake.NewClientBuilder().Build(),
				testNamespace,
				testCase.controllerName,
				testCase.leaseDuration,
			).(*renewer)
			testCase.assertions(t, r)
		})
	}

	t.Run("non-positive lease duration panics", func(t *testing.T) {
		for _, d := range []time.Duration{0, -1 * time.Second} {
			require.Panics(t, func() {
				NewRenewer(
					fake.NewClientBuilder().Build(),
					testNamespace,
					"alpha",
					d,
				)
			})
		}
	})
}

func TestRenewer_Start(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, coordinationv1.AddToScheme(scheme))

	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := &renewer{
		client:         c,
		namespace:      "kargo",
		controllerName: "alpha",
		leaseName:      "kargo-controller-alpha",
		holderIdentity: "test-holder",
		leaseDuration:  30 * time.Second,
		renewInterval:  10 * time.Millisecond, // Short interval to speed up the test
	}

	objKey := types.NamespacedName{Namespace: r.namespace, Name: r.leaseName}

	// Verify a lease is created after Start is called.
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- r.Start(ctx) }()
	require.Eventually(
		t,
		func() bool {
			lease := &coordinationv1.Lease{}
			return c.Get(t.Context(), objKey, lease) == nil
		},
		time.Second,
		10*time.Millisecond,
	)
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancel")
	}

	// Verify the lease was deleted on shutdown.
	lease := &coordinationv1.Lease{}
	err := c.Get(t.Context(), objKey, lease)
	require.True(t, apierrors.IsNotFound(err))
}

func TestRenewer_renew(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, coordinationv1.AddToScheme(scheme))

	const testNamespace = "kargo"
	const testLeaseName = "fake-lease"
	const testHolder = "test-holder"

	testLeaseDuration := 30 * time.Second

	objKey := types.NamespacedName{Namespace: testNamespace, Name: testLeaseName}

	old := time.Now().Add(-1 * time.Hour)

	testCases := []struct {
		name           string
		client         client.Client
		controllerName string
		assertions     func(*testing.T, client.Client)
	}{
		{
			name:           "creates lease when absent",
			client:         fake.NewClientBuilder().WithScheme(scheme).Build(),
			controllerName: "fake-controller",
			assertions: func(t *testing.T, c client.Client) {
				lease := &coordinationv1.Lease{}
				err := c.Get(t.Context(), objKey, lease)
				require.NoError(t, err)
				require.Equal(t, "fake-controller", lease.Labels[kargoapi.LabelKeyController])
				require.NotNil(t, lease.Spec.HolderIdentity)
				require.Equal(t, testHolder, *lease.Spec.HolderIdentity)
				require.NotNil(t, lease.Spec.LeaseDurationSeconds)
				require.Equal(
					t,
					int32(testLeaseDuration.Seconds()),
					*lease.Spec.LeaseDurationSeconds,
				)
				require.NotNil(t, lease.Spec.RenewTime)
				require.NotNil(t, lease.Spec.AcquireTime)
			},
		},
		{
			name:           "updates existing lease and preserves AcquireTime",
			controllerName: "fake-controller",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNamespace,
						Name:      testLeaseName,
						Labels:    map[string]string{kargoapi.LabelKeyController: "fake-controller"},
					},
					Spec: coordinationv1.LeaseSpec{
						RenewTime:   &metav1.MicroTime{Time: old},
						AcquireTime: &metav1.MicroTime{Time: old},
					},
				},
			).Build(),
			assertions: func(t *testing.T, c client.Client) {
				lease := &coordinationv1.Lease{}
				err := c.Get(t.Context(), objKey, lease)
				require.NoError(t, err)
				// RenewTime should have advanced but AcquireTime should be preserved
				// across renewals. metav1.MicroTime's protobuf path truncates to
				// microseconds, so allow a 1µs tolerance.
				require.NotNil(t, lease.Spec.RenewTime)
				require.True(t, lease.Spec.RenewTime.After(old))
				require.NotNil(t, lease.Spec.AcquireTime)
				// WithinDuration accounts for possible loss of precision in AcquireTime
				// due quirks of the fake client.
				require.WithinDuration(
					t,
					old,
					lease.Spec.AcquireTime.Time,
					time.Second,
				)
			},
		},
		{
			name:           "unnamed controller writes empty label value",
			client:         fake.NewClientBuilder().WithScheme(scheme).Build(),
			controllerName: "",
			assertions: func(t *testing.T, c client.Client) {
				lease := &coordinationv1.Lease{}
				err := c.Get(t.Context(), objKey, lease)
				require.NoError(t, err)
				// Label should exist, but its value should be an empty string.
				value, exists := lease.Labels[kargoapi.LabelKeyController]
				require.True(t, exists)
				require.Empty(t, value)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := &renewer{
				client:         testCase.client,
				controllerName: testCase.controllerName,
				namespace:      testNamespace,
				leaseName:      testLeaseName,
				holderIdentity: testHolder,
				leaseDuration:  testLeaseDuration,
				renewInterval:  testLeaseDuration / 3,
			}
			err := r.renew(t.Context())
			require.NoError(t, err)
			testCase.assertions(t, testCase.client)
		})
	}
}

func TestRenewer_delete(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, coordinationv1.AddToScheme(scheme))

	objKey := types.NamespacedName{Namespace: "kargo", Name: "kargo-controller-alpha"}

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, client.Client, error)
	}{
		{
			name: "removes lease",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: objKey.Namespace,
						Name:      objKey.Name,
					},
				},
			).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				lease := &coordinationv1.Lease{}
				err = c.Get(t.Context(), objKey, lease)
				require.True(t, apierrors.IsNotFound(err))
			},
		},
		{
			name:   "is idempotent when lease is already absent",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				// Deleting a non-existent lease should not return an error.
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := &renewer{
				client:         testCase.client,
				controllerName: "alpha",
				namespace:      "kargo",
				leaseName:      "kargo-controller-alpha",
			}
			err := r.delete(t.Context())
			testCase.assertions(t, testCase.client, err)
		})
	}
}
