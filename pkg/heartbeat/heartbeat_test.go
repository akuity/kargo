package heartbeat

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetAll(t *testing.T) {
	now := time.Now()
	freshTime := now.Add(-5 * time.Second)
	staleTime := now.Add(-1 * time.Hour)
	duration := int32(30)

	testCases := []struct {
		name       string
		objects    []client.Object
		assertions func(*testing.T, map[string]Heartbeat, error)
	}{
		{
			name: "no leases → empty result",
			assertions: func(t *testing.T, got map[string]Heartbeat, err error) {
				require.NoError(t, err)
				require.Empty(t, got)
			},
		},
		{
			name: "returns alive and dead controllers indexed by controller name",
			objects: []client.Object{
				newLease("kargo", "kargo-controller-alpha", "alpha", &freshTime, &duration),
				newLease("kargo", "kargo-controller-beta", "beta", &staleTime, &duration),
			},
			assertions: func(t *testing.T, got map[string]Heartbeat, err error) {
				require.NoError(t, err)
				require.Len(t, got, 2)
				require.Equal(t, StatusAlive, got["alpha"].Status)
				require.Equal(t, "alpha", got["alpha"].Controller)
				require.Equal(t, StatusDead, got["beta"].Status)
				require.Equal(t, "beta", got["beta"].Controller)
			},
		},
		{
			name: "unnamed controller is keyed by empty string",
			objects: []client.Object{
				newLease("kargo", "kargo-controller-unnamed", "", &freshTime, &duration),
			},
			assertions: func(t *testing.T, got map[string]Heartbeat, err error) {
				require.NoError(t, err)
				require.Len(t, got, 1)
				hb, ok := got[""]
				require.True(t, ok,
					"unnamed controller heartbeat must be keyed by empty string")
				require.Equal(t, StatusAlive, hb.Status)
				require.Empty(t, hb.Controller)
			},
		},
		{
			name: "leases without the controller label are excluded",
			objects: []client.Object{
				newLease("kargo", "kargo-controller-alpha", "alpha", &freshTime, &duration),
				&coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo",
						Name:      "unrelated",
					},
					Spec: coordinationv1.LeaseSpec{
						RenewTime:            &metav1.MicroTime{Time: freshTime},
						LeaseDurationSeconds: &duration,
					},
				},
			},
			assertions: func(t *testing.T, got map[string]Heartbeat, err error) {
				require.NoError(t, err)
				require.Len(t, got, 1)
				_, alphaPresent := got["alpha"]
				require.True(t, alphaPresent)
			},
		},
		{
			name: "leases in other namespaces are excluded",
			objects: []client.Object{
				newLease("kargo", "kargo-controller-alpha", "alpha", &freshTime, &duration),
				newLease("other", "kargo-controller-stranger", "stranger", &freshTime, &duration),
			},
			assertions: func(t *testing.T, got map[string]Heartbeat, err error) {
				require.NoError(t, err)
				require.Len(t, got, 1)
				_, alphaPresent := got["alpha"]
				require.True(t, alphaPresent)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := newFakeClient(t, testCase.objects...)
			got, err := GetAll(t.Context(), c, "kargo")
			testCase.assertions(t, got, err)
		})
	}
}

func TestLeaseToHeartbeat(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	duration := int32(30)

	testCases := []struct {
		name           string
		controllerName string
		spec           coordinationv1.LeaseSpec
		assertions     func(*testing.T, Heartbeat)
	}{
		{
			name:           "renewTime missing → dead with no Timestamp",
			controllerName: "foo",
			spec:           coordinationv1.LeaseSpec{LeaseDurationSeconds: &duration},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Equal(t, "foo", hb.Controller)
				require.Equal(t, StatusDead, hb.Status)
				require.Nil(t, hb.Timestamp)
			},
		},
		{
			name:           "leaseDurationSeconds missing → dead with Timestamp",
			controllerName: "foo",
			spec: coordinationv1.LeaseSpec{
				RenewTime: &metav1.MicroTime{Time: now.Add(-1 * time.Second)},
			},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Equal(t, StatusDead, hb.Status)
				require.NotNil(t, hb.Timestamp)
			},
		},
		{
			name:           "fresh renewTime → alive",
			controllerName: "foo",
			spec: coordinationv1.LeaseSpec{
				RenewTime:            &metav1.MicroTime{Time: now.Add(-5 * time.Second)},
				LeaseDurationSeconds: &duration,
			},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Equal(t, StatusAlive, hb.Status)
				require.NotNil(t, hb.Timestamp)
			},
		},
		{
			name:           "renewTime exactly at expiry → dead",
			controllerName: "foo",
			spec: coordinationv1.LeaseSpec{
				RenewTime: &metav1.MicroTime{
					Time: now.Add(-time.Duration(duration) * time.Second),
				},
				LeaseDurationSeconds: &duration,
			},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Equal(t, StatusDead, hb.Status)
			},
		},
		{
			name:           "stale renewTime → dead",
			controllerName: "foo",
			spec: coordinationv1.LeaseSpec{
				RenewTime:            &metav1.MicroTime{Time: now.Add(-1 * time.Hour)},
				LeaseDurationSeconds: &duration,
			},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Equal(t, StatusDead, hb.Status)
			},
		},
		{
			name:           "empty controller name is preserved on the heartbeat",
			controllerName: "",
			spec: coordinationv1.LeaseSpec{
				RenewTime:            &metav1.MicroTime{Time: now.Add(-1 * time.Second)},
				LeaseDurationSeconds: &duration,
			},
			assertions: func(t *testing.T, hb Heartbeat) {
				require.Empty(t, hb.Controller)
				require.Equal(t, StatusAlive, hb.Status)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			lease := &coordinationv1.Lease{Spec: testCase.spec}
			testCase.assertions(t, leaseToHeartbeat(testCase.controllerName, lease, now))
		})
	}
}

// newFakeClient builds a controller-runtime fake client whose scheme knows
// about Leases and is preloaded with the given objects.
func newFakeClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, coordinationv1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

// newLease constructs a Lease bearing the controller label, suitable for
// seeding a fake client. nil renewTime omits the field; nil
// leaseDurationSeconds omits the field.
func newLease(
	namespace, name, controllerName string,
	renewTime *time.Time,
	leaseDurationSeconds *int32,
) *coordinationv1.Lease {
	spec := coordinationv1.LeaseSpec{LeaseDurationSeconds: leaseDurationSeconds}
	if renewTime != nil {
		spec.RenewTime = &metav1.MicroTime{Time: *renewTime}
	}
	return &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    map[string]string{kargoapi.LabelKeyController: controllerName},
		},
		Spec: spec,
	}
}
