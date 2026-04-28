package heartbeat

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetAll(t *testing.T) {
	const testNamespace = "kargo"

	scheme := runtime.NewScheme()
	require.NoError(t, coordinationv1.AddToScheme(scheme))

	now := time.Now()
	freshTime := now.Add(-5 * time.Second)
	staleTime := now.Add(-1 * time.Hour)

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, map[string]Heartbeat, error)
	}{
		{
			name:   "no leases → empty result",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, got map[string]Heartbeat, err error) {
				require.NoError(t, err)
				require.Empty(t, got)
			},
		},
		{
			name: "returns alive and dead controllers indexed by controller name",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNamespace,
						Name:      "fake-lease-1",
						Labels:    map[string]string{kargoapi.LabelKeyController: "fake-1"},
					},
					Spec: coordinationv1.LeaseSpec{
						RenewTime:            &metav1.MicroTime{Time: freshTime},
						LeaseDurationSeconds: ptr.To(int32(30)),
					},
				},
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNamespace,
						Name:      "fake-lease-2",
						Labels:    map[string]string{kargoapi.LabelKeyController: "fake-2"},
					},
					Spec: coordinationv1.LeaseSpec{
						RenewTime:            &metav1.MicroTime{Time: staleTime},
						LeaseDurationSeconds: ptr.To(int32(30)),
					},
				},
			).Build(),
			assertions: func(t *testing.T, heartbeats map[string]Heartbeat, err error) {
				require.NoError(t, err)
				require.Len(t, heartbeats, 2)
				require.Equal(t, StatusAlive, heartbeats["fake-1"].Status)
				require.Equal(t, "fake-1", heartbeats["fake-1"].Controller)
				require.Equal(t, StatusDead, heartbeats["fake-2"].Status)
				require.Equal(t, "fake-2", heartbeats["fake-2"].Controller)
			},
		},
		{
			name: "unnamed controller is keyed by empty string",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNamespace,
						Name:      "fake-lease",
						Labels:    map[string]string{kargoapi.LabelKeyController: ""},
					},
					Spec: coordinationv1.LeaseSpec{
						RenewTime: &metav1.MicroTime{Time: freshTime},
					},
				},
			).Build(),
			assertions: func(t *testing.T, heartbeats map[string]Heartbeat, err error) {
				require.NoError(t, err)
				require.Len(t, heartbeats, 1)
				hb, exists := heartbeats[""]
				require.True(t, exists)
				require.Empty(t, hb.Controller)
			},
		},
		{
			name: "leases without the controller label are excluded",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNamespace,
						Name:      "fake-lease-1",
						Labels:    map[string]string{kargoapi.LabelKeyController: "fake-1"},
					},
				},
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNamespace,
						Name:      "fake-lease-2",
					},
				},
			).Build(),
			assertions: func(t *testing.T, heartbeats map[string]Heartbeat, err error) {
				require.NoError(t, err)
				require.Len(t, heartbeats, 1)
				_, exists := heartbeats["fake-1"]
				require.True(t, exists)
			},
		},
		{
			name: "leases in other namespaces are excluded",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNamespace,
						Name:      "fake-lease-1",
						Labels:    map[string]string{kargoapi.LabelKeyController: "fake-1"},
					},
				},
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "other-namespace",
						Name:      "fake-lease-2",
						Labels:    map[string]string{kargoapi.LabelKeyController: "fake-2"},
					},
				},
			).Build(),
			assertions: func(t *testing.T, got map[string]Heartbeat, err error) {
				require.NoError(t, err)
				require.Len(t, got, 1)
				_, exists := got["fake-1"]
				require.True(t, exists)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			heartbeats, err := GetAll(t.Context(), testCase.client, testNamespace)
			testCase.assertions(t, heartbeats, err)
		})
	}
}

func TestLeaseToHeartbeat(t *testing.T) {
	const testControllerName = "fake-controller"

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	duration := int32(30)

	testCases := []struct {
		name       string
		lease      *coordinationv1.Lease
		assertions func(*testing.T, Heartbeat)
	}{
		{
			name: "renewTime missing → dead with no Timestamp",
			lease: &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{kargoapi.LabelKeyController: testControllerName},
				},
				Spec: coordinationv1.LeaseSpec{LeaseDurationSeconds: &duration},
			},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Equal(t, testControllerName, hb.Controller)
				require.Equal(t, StatusDead, hb.Status)
				require.Nil(t, hb.Timestamp)
			},
		},
		{
			name: "leaseDurationSeconds missing → dead with Timestamp",
			lease: &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{kargoapi.LabelKeyController: testControllerName},
				},
				Spec: coordinationv1.LeaseSpec{
					RenewTime: &metav1.MicroTime{Time: now.Add(-1 * time.Second)},
				},
			},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Equal(t, StatusDead, hb.Status)
				require.NotNil(t, hb.Timestamp)
			},
		},
		{
			name: "fresh renewTime → alive",
			lease: &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{kargoapi.LabelKeyController: testControllerName},
				},
				Spec: coordinationv1.LeaseSpec{
					RenewTime:            &metav1.MicroTime{Time: now.Add(-5 * time.Second)},
					LeaseDurationSeconds: &duration,
				},
			},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Equal(t, StatusAlive, hb.Status)
				require.NotNil(t, hb.Timestamp)
			},
		},
		{
			name: "renewTime exactly at expiry → dead",
			lease: &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{kargoapi.LabelKeyController: testControllerName},
				},
				Spec: coordinationv1.LeaseSpec{
					RenewTime:            &metav1.MicroTime{Time: now.Add(-time.Duration(duration) * time.Second)},
					LeaseDurationSeconds: &duration,
				},
			},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Equal(t, StatusDead, hb.Status)
			},
		},
		{
			name: "stale renewTime → dead",
			lease: &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{kargoapi.LabelKeyController: testControllerName},
				},
				Spec: coordinationv1.LeaseSpec{
					RenewTime:            &metav1.MicroTime{Time: now.Add(-1 * time.Hour)},
					LeaseDurationSeconds: &duration,
				},
			},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Equal(t, StatusDead, hb.Status)
			},
		},
		{
			name: "empty controller name is preserved on the heartbeat",
			lease: &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{kargoapi.LabelKeyController: ""},
				},
				Spec: coordinationv1.LeaseSpec{
					RenewTime:            &metav1.MicroTime{Time: now.Add(-1 * time.Hour)},
					LeaseDurationSeconds: &duration,
				},
			},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Empty(t, hb.Controller)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				leaseToHeartbeat(testCase.lease, now),
			)
		})
	}
}
