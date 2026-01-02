package api

import (
	"context"
	"testing"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func generatePromotionWindow(name, kind, schedule, duration string) *kargoapi.PromotionWindow {
	return &kargoapi.PromotionWindow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
		},
		Spec: kargoapi.PromotionWindowSpec{
			Kind:     kind,
			Schedule: schedule,
			Duration: duration,
		},
	}
}

func TestGetPromotionWindowSpec(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))
	namespace := "test"

	tests := []struct {
		name       string
		ref        kargoapi.PromotionWindowReference
		client     client.Client
		project    string
		assertions func(t *testing.T, spec *kargoapi.PromotionWindowSpec, err error)
	}{
		{
			name:    "reject empty promotion window reference",
			ref:     kargoapi.PromotionWindowReference{},
			client:  fake.NewClientBuilder().WithScheme(scheme).Build(),
			project: namespace,
			assertions: func(t *testing.T, spec *kargoapi.PromotionWindowSpec, err error) {
				require.Error(t, err)
				require.Nil(t, spec)
				require.Contains(t, err.Error(), "missing promotion window reference")
			},
		},
		{
			name:    "reject nil client",
			ref:     kargoapi.PromotionWindowReference{Name: "x", Kind: "PromotionWindow"},
			client:  nil,
			project: namespace,
			assertions: func(t *testing.T, spec *kargoapi.PromotionWindowSpec, err error) {
				require.Error(t, err)
				require.Nil(t, spec)
			},
		},
		{
			name:    "reject empty project",
			ref:     kargoapi.PromotionWindowReference{Name: "x", Kind: "PromotionWindow"},
			client:  fake.NewClientBuilder().WithScheme(scheme).Build(),
			project: "",
			assertions: func(t *testing.T, spec *kargoapi.PromotionWindowSpec, err error) {
				require.Error(t, err)
				require.Nil(t, spec)
			},
		},
		{
			name:    "accept empty kind in promotion window reference",
			ref:     kargoapi.PromotionWindowReference{Name: "test-window"},
			project: namespace,
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				generatePromotionWindow("test-window", "allow", "0 0 * * *", "1h"),
			).Build(),
			assertions: func(t *testing.T, spec *kargoapi.PromotionWindowSpec, err error) {
				require.NoError(t, err)
				require.NotNil(t, spec)
				require.Equal(t, "allow", spec.Kind)
				require.Equal(t, "0 0 * * *", spec.Schedule)
				require.Equal(t, "1h", spec.Duration)
			},
		},
		{
			name:    "error when promotion window not found",
			ref:     kargoapi.PromotionWindowReference{Name: "missing", Kind: "PromotionWindow"},
			client:  fake.NewClientBuilder().WithScheme(scheme).Build(),
			project: namespace,
			assertions: func(t *testing.T, spec *kargoapi.PromotionWindowSpec, err error) {
				require.Error(t, err)
				require.Nil(t, spec)
			},
		},
		{
			name:    "error when unknown kind in promotion window reference",
			ref:     kargoapi.PromotionWindowReference{Name: "x", Kind: "SomeOtherKind"},
			client:  fake.NewClientBuilder().WithScheme(scheme).Build(),
			project: namespace,
			assertions: func(t *testing.T, spec *kargoapi.PromotionWindowSpec, err error) {
				require.Error(t, err)
				require.Nil(t, spec)
			},
		},
		{
			name:    "error when name empty in promotion window reference",
			ref:     kargoapi.PromotionWindowReference{Name: "", Kind: "PromotionWindow"},
			client:  fake.NewClientBuilder().WithScheme(scheme).Build(),
			project: namespace,
			assertions: func(t *testing.T, spec *kargoapi.PromotionWindowSpec, err error) {
				require.Error(t, err)
				require.Nil(t, spec)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := getPromotionWindowSpec(context.Background(), tt.ref, tt.client, tt.project)
			tt.assertions(t, spec, err)
		})
	}
}

func TestCheckPromotionWindows(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))
	namespace := "test"
	janPastMidnight := time.Date(2024, 1, 1, 0, 30, 0, 0, time.UTC) // Jan 1, 2024 00:30 UTC

	tests := []struct {
		name       string
		now        time.Time
		client     client.Client
		windowRefs []kargoapi.PromotionWindowReference
		assertions func(t *testing.T, allow bool, err error)
	}{
		{
			name:       "allow promotions by default",
			now:        janPastMidnight, // Jan 1, 2024 00:30 UTC
			client:     fake.NewClientBuilder().WithScheme(scheme).Build(),
			windowRefs: []kargoapi.PromotionWindowReference{},
			assertions: func(t *testing.T, allow bool, err error) {
				require.NoError(t, err)
				require.True(t, allow)
			},
		},
		{
			name: "allow promotion with active allow window",
			now:  janPastMidnight, // Jan 1, 2024 00:30 UTC
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				generatePromotionWindow("allow-window", "allow", "0 0 * * *", "1h"),
			).Build(),
			windowRefs: []kargoapi.PromotionWindowReference{{Name: "allow-window", Kind: "PromotionWindow"}},
			assertions: func(t *testing.T, allow bool, err error) {
				require.NoError(t, err)
				require.True(t, allow)
			},
		},
		{
			name: "allow promotion when at least one allow window is active",
			now:  janPastMidnight, // Jan 1, 2024 00:30 UTC
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				generatePromotionWindow("allow-window-1", "allow", "0 3 * * *", "1h"),
				generatePromotionWindow("allow-window-2", "allow", "0 0 * * *", "1h"),
			).Build(),
			windowRefs: []kargoapi.PromotionWindowReference{
				{Name: "allow-window-1", Kind: "PromotionWindow"},
				{Name: "allow-window-2", Kind: "PromotionWindow"},
			},
			assertions: func(t *testing.T, allow bool, err error) {
				require.NoError(t, err)
				require.True(t, allow)
			},
		},
		{
			name: "disallow promotion on inactive allow window",
			now:  time.Date(2024, 1, 1, 1, 30, 0, 0, time.UTC), // Jan 1, 2024 01:30 UTC
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				generatePromotionWindow("allow-window", "allow", "0 0 * * *", "1h"),
			).Build(),
			windowRefs: []kargoapi.PromotionWindowReference{{Name: "allow-window", Kind: "PromotionWindow"}},
			assertions: func(t *testing.T, allow bool, err error) {
				require.NoError(t, err)
				require.False(t, allow)
			},
		},
		{
			name: "disallow promotion on inactive allow windows",
			now:  time.Date(2024, 1, 1, 1, 30, 0, 0, time.UTC), // Jan 1, 2024 01:30 UTC
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				generatePromotionWindow("allow-window-1", "allow", "0 0 * * *", "1h"),
				generatePromotionWindow("allow-window-2", "allow", "0 0 * * *", "1h"),
			).Build(),
			windowRefs: []kargoapi.PromotionWindowReference{
				{Name: "allow-window-1", Kind: "PromotionWindow"},
				{Name: "allow-window-2", Kind: "PromotionWindow"},
			},
			assertions: func(t *testing.T, allow bool, err error) {
				require.NoError(t, err)
				require.False(t, allow)
			},
		},
		{
			name: "deny promotion on active deny window",
			now:  janPastMidnight, // Jan 1, 2024 00:30 UTC
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				generatePromotionWindow("deny-window", "deny", "0 0 * * *", "1h"),
			).Build(),
			windowRefs: []kargoapi.PromotionWindowReference{{Name: "deny-window", Kind: "PromotionWindow"}},
			assertions: func(t *testing.T, allow bool, err error) {
				require.NoError(t, err)
				require.False(t, allow)
			},
		},
		{
			name: "allow promotion on inactive deny window",
			now:  janPastMidnight, // Jan 1, 2024 00:30 UTC
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				generatePromotionWindow("deny-window", "deny", "0 3 * * *", "1h"),
			).Build(),
			windowRefs: []kargoapi.PromotionWindowReference{{Name: "deny-window", Kind: "PromotionWindow"}},
			assertions: func(t *testing.T, allow bool, err error) {
				require.NoError(t, err)
				require.True(t, allow)
			},
		},
		{
			name: "disallow promotion with deny window active and allow window inactive",
			now:  time.Date(2024, 1, 1, 0, 30, 0, 0, time.UTC), // Jan 1, 2024 00:30 UTC
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				generatePromotionWindow("deny-window", "deny", "0 0 * * *", "1h"),
				generatePromotionWindow("allow-window", "allow", "0 2 * * *", "1h"),
			).Build(),
			windowRefs: []kargoapi.PromotionWindowReference{
				{Name: "deny-window", Kind: "PromotionWindow"},
				{Name: "allow-window", Kind: "PromotionWindow"},
			},
			assertions: func(t *testing.T, allow bool, err error) {
				require.NoError(t, err)
				require.False(t, allow)
			},
		},
		{
			name: "disallow promotion with both deny window and allow window active",
			now:  janPastMidnight, // Jan 1, 2024 00:30 UTC
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				generatePromotionWindow("deny-window", "deny", "0 0 * * *", "1h"),
				generatePromotionWindow("allow-window", "allow", "0 0 * * *", "1h"),
			).Build(),
			windowRefs: []kargoapi.PromotionWindowReference{
				{Name: "deny-window", Kind: "PromotionWindow"},
				{Name: "allow-window", Kind: "PromotionWindow"},
			},
			assertions: func(t *testing.T, allow bool, err error) {
				require.NoError(t, err)
				require.False(t, allow)
			},
		},
		{
			name: "disallow promotion with both allow window and deny window active",
			now:  janPastMidnight, // Jan 1, 2024 00:30 UTC
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				generatePromotionWindow("allow-window", "allow", "0 0 * * *", "1h"),
				generatePromotionWindow("deny-window", "deny", "0 0 * * *", "1h"),
			).Build(),
			windowRefs: []kargoapi.PromotionWindowReference{
				{Name: "allow-window", Kind: "PromotionWindow"},
				{Name: "deny-window", Kind: "PromotionWindow"},
			},
			assertions: func(t *testing.T, allow bool, err error) {
				require.NoError(t, err)
				require.False(t, allow)
			},
		},
		{
			name: "allow promotion with allow window active and deny window inactive",
			now:  janPastMidnight, // Jan 1, 2024 00:30 UTC
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				generatePromotionWindow("allow-window", "allow", "0 0 * * *", "1h"),
				generatePromotionWindow("deny-window", "deny", "0 2 * * *", "1h"),
			).Build(),
			windowRefs: []kargoapi.PromotionWindowReference{
				{Name: "allow-window", Kind: "PromotionWindow"},
				{Name: "deny-window", Kind: "PromotionWindow"},
			},
			assertions: func(t *testing.T, allow bool, err error) {
				require.NoError(t, err)
				require.True(t, allow)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			active, err := CheckPromotionWindows(
				context.Background(),
				tt.now,
				tt.windowRefs,
				tt.client,
				namespace,
			)

			tt.assertions(t, active, err)
		})
	}
}
