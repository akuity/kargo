package projectconfigstore

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestToAPI(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	createdAt := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 11, 23, 11, 0, 0, 123456000, time.UTC)
	conditions := []metav1.Condition{{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		LastTransitionTime: metav1.NewTime(createdAt),
	}}
	conditionsJSON, err := json.Marshal(conditions)
	require.NoError(t, err)
	observed := int64(2)
	handled := "handled-token"
	token := "requested-token"
	newConfig := func() Config {
		return Config{
			ProjectConfigRow: database.ProjectConfigRow{
				ID:          id,
				ProjectName: "my-project",
				ProjectUID:  "project-uid",
				Labels:      map[string]string{"team": "a"},
				Annotations: map[string]string{"note": "x"},
				Spec: json.RawMessage(
					`{"promotionPolicies":[{"stage":"dev","autoPromotionEnabled":true}]}`,
				),
				Generation:               3,
				StatusConditions:         conditionsJSON,
				StatusObservedGeneration: &observed,
				StatusLastHandledRefresh: &handled,
				CreatedAt:                createdAt,
				UpdatedAt:                updatedAt,
			},
			Receivers: []database.ProjectConfigWebhookReceiverRow{
				{ProjectConfigID: id, Name: "zeta", Path: "/z", URL: "https://x/z"},
				{ProjectConfigID: id, Name: "alpha", Path: "/a", URL: "https://x/a"},
			},
		}
	}
	testCases := []struct {
		name   string
		config func() Config
		assert func(*testing.T, Config, *kargoapi.ProjectConfig)
	}{
		{
			name:   "full row",
			config: newConfig,
			assert: func(t *testing.T, _ Config, cfg *kargoapi.ProjectConfig) {
				require.Equal(t, "my-project", cfg.Name)
				require.Equal(t, "my-project", cfg.Namespace)
				require.Equal(t, types.UID(id.String()), cfg.UID)
				require.Equal(t, "1795431600123456", cfg.ResourceVersion)
				require.Equal(t, int64(3), cfg.Generation)
				require.True(t, cfg.CreationTimestamp.Time.Equal(createdAt))
				require.Equal(t, map[string]string{"team": "a"}, cfg.Labels)
				require.Equal(t, map[string]string{"note": "x"}, cfg.Annotations)
				require.Equal(t, kargoapi.ProjectConfigSpec{
					PromotionPolicies: []kargoapi.PromotionPolicy{{
						Stage:                "dev",
						AutoPromotionEnabled: true,
					}},
				}, cfg.Spec)
				require.Len(t, cfg.Status.Conditions, 1)
				require.Equal(t, kargoapi.ConditionTypeReady, cfg.Status.Conditions[0].Type)
				require.Equal(t, metav1.ConditionTrue, cfg.Status.Conditions[0].Status)
				require.Equal(t, int64(2), cfg.Status.ObservedGeneration)
				require.Equal(t, "handled-token", cfg.Status.LastHandledRefresh)
				require.Equal(t, []kargoapi.WebhookReceiverDetails{
					{Name: "alpha", Path: "/a", URL: "https://x/a"},
					{Name: "zeta", Path: "/z", URL: "https://x/z"},
				}, cfg.Status.WebhookReceivers)
			},
		},
		{
			name: "refresh token served as the refresh annotation",
			config: func() Config {
				c := newConfig()
				c.RefreshToken = &token
				return c
			},
			assert: func(t *testing.T, in Config, cfg *kargoapi.ProjectConfig) {
				require.Equal(t, map[string]string{
					"note":                        "x",
					kargoapi.AnnotationKeyRefresh: "requested-token",
				}, cfg.Annotations)
				// The row's map is left alone.
				require.Equal(t, map[string]string{"note": "x"}, in.Annotations)
			},
		},
		{
			name: "refresh token with no other annotations",
			config: func() Config {
				c := newConfig()
				c.Annotations = nil
				c.RefreshToken = &token
				return c
			},
			assert: func(t *testing.T, _ Config, cfg *kargoapi.ProjectConfig) {
				require.Equal(t, map[string]string{
					kargoapi.AnnotationKeyRefresh: "requested-token",
				}, cfg.Annotations)
			},
		},
		{
			name: "empty status and metadata",
			config: func() Config {
				return Config{ProjectConfigRow: database.ProjectConfigRow{
					ID:          id,
					ProjectName: "my-project",
					Labels:      map[string]string{},
					Annotations: map[string]string{},
					Spec:        json.RawMessage(`{}`),
					Generation:  1,
				}}
			},
			assert: func(t *testing.T, _ Config, cfg *kargoapi.ProjectConfig) {
				require.Nil(t, cfg.Labels)
				require.Nil(t, cfg.Annotations)
				require.Equal(t, kargoapi.ProjectConfigSpec{}, cfg.Spec)
				require.Equal(t, kargoapi.ProjectConfigStatus{}, cfg.Status)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			in := testCase.config()
			testCase.assert(t, in, ToAPI(in))
		})
	}
}

func TestSpecColumns(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		config string
		assert func(t *testing.T, labels, annotations, spec []byte, err error)
	}{
		{
			name: "unknown spec fields dropped",
			config: `{
				"metadata": {"name": "p", "namespace": "p"},
				"spec": {
					"bogus": true,
					"webhookReceivers": [{
						"name": "gh",
						"extra": 1,
						"github": {"secretRef": {"name": "s"}}
					}]
				}
			}`,
			assert: func(t *testing.T, _, _, spec []byte, err error) {
				require.NoError(t, err)
				require.JSONEq(
					t,
					`{"webhookReceivers":[{"name":"gh","github":{"secretRef":{"name":"s"}}}]}`,
					string(spec),
				)
			},
		},
		{
			name: "metadata kept and dropped",
			config: `{
				"metadata": {
					"name": "p",
					"namespace": "p",
					"labels": {"team": "a"},
					"annotations": {
						"kargo.akuity.io/refresh": "t",
						"kubectl.kubernetes.io/last-applied-configuration": "{}",
						"argocd.argoproj.io/tracking-id": "app:kargo.akuity.io/ProjectConfig:p/p",
						"note": "x"
					},
					"finalizers": ["kargo.akuity.io/finalizer"],
					"ownerReferences": [{
						"apiVersion": "v1",
						"kind": "Namespace",
						"name": "p",
						"uid": "u"
					}]
				},
				"spec": {}
			}`,
			assert: func(t *testing.T, labels, annotations, spec []byte, err error) {
				require.NoError(t, err)
				require.JSONEq(t, `{"team":"a"}`, string(labels))
				require.JSONEq(t, `{
					"kubectl.kubernetes.io/last-applied-configuration": "{}",
					"argocd.argoproj.io/tracking-id": "app:kargo.akuity.io/ProjectConfig:p/p",
					"note": "x"
				}`, string(annotations))
				require.JSONEq(t, `{}`, string(spec))
			},
		},
		{
			name:   "absent metadata maps become empty objects",
			config: `{"metadata": {"name": "p"}}`,
			assert: func(t *testing.T, labels, annotations, spec []byte, err error) {
				require.NoError(t, err)
				require.JSONEq(t, `{}`, string(labels))
				require.JSONEq(t, `{}`, string(annotations))
				require.JSONEq(t, `{}`, string(spec))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg := &kargoapi.ProjectConfig{}
			require.NoError(t, json.Unmarshal([]byte(testCase.config), cfg))
			before := cfg.DeepCopy()
			labels, annotations, spec, err := specColumns(cfg)
			testCase.assert(t, labels, annotations, spec, err)
			// The argument is left alone.
			require.Equal(t, before, cfg)
		})
	}
}

func TestStatusConditions(t *testing.T) {
	t.Parallel()
	conditions, err := statusConditions(nil)
	require.NoError(t, err)
	require.Nil(t, conditions)

	conditions, err = statusConditions([]metav1.Condition{{
		Type:   kargoapi.ConditionTypeReady,
		Status: metav1.ConditionFalse,
		Reason: "Bad",
	}})
	require.NoError(t, err)
	var decoded []metav1.Condition
	require.NoError(t, json.Unmarshal(conditions, &decoded))
	require.Len(t, decoded, 1)
	require.Equal(t, "Bad", decoded[0].Reason)
}
