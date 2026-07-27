// Package fidelity proves (or disproves) wire compatibility between the
// openapi-generator-generated Go client (../generated) and the canonical
// serialization produced by the real Kargo/Kubernetes API types.
//
// Method: construct richly-populated resources using the canonical types
// (github.com/akuity/kargo/api/v1alpha1 and k8s.io/api), marshal them with
// encoding/json -- byte-for-byte what the Kargo API server sends -- then
// unmarshal into the generated model and re-marshal. Round-trip output must
// be JSON-equal to ground truth.
//
// This spike also surfaced spec-infidelity defects -- swagger.json modeled
// Quantity, IntOrString, MicroTime, and arbitrary-JSON values structurally
// because swag reflects Go structs and ignores custom MarshalJSON. Those have
// since been fixed in the canonical spec (.swaggo overrides plus
// fix-swagger-spec.sh Pass 4); the TestWireFormat_* tests pin the corrected
// behavior in both clients.
package fidelity

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	gen "github.com/akuity/kargo/pkg/x/client/generated"
)

// roundTrip marshals the canonical object (ground truth: exactly what the
// API server sends), unmarshals into the generated model, re-marshals, and
// requires the result to be JSON-equal to ground truth.
func roundTrip(t *testing.T, canonical any, model any) {
	t.Helper()
	truth, err := json.Marshal(canonical)
	require.NoError(t, err, "marshaling canonical ground truth")
	require.NoError(
		t, json.Unmarshal(truth, model),
		"unmarshaling ground truth into generated model:\n%s", truth,
	)
	replay, err := json.Marshal(model)
	require.NoError(t, err, "re-marshaling generated model")
	require.JSONEq(t, string(truth), string(replay))
}

var testTime = metav1.NewTime(
	time.Date(2026, 7, 16, 12, 30, 45, 0, time.UTC),
)

func testObjectMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:              name,
		Namespace:         "kargo-demo",
		UID:               "3a3b7e30-1f0e-4bda-a651-8a0b41d0f2c6",
		ResourceVersion:   "123456",
		Generation:        7,
		CreationTimestamp: testTime,
		Labels: map[string]string{
			"app.kubernetes.io/part-of": "kargo-demo",
		},
		Annotations: map[string]string{
			"kargo.akuity.io/color": "blue",
		},
		Finalizers: []string{"kargo.akuity.io/finalizer"},
	}
}

func TestStageRoundTrip(t *testing.T) {
	stage := &kargoapi.Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Stage",
		},
		ObjectMeta: testObjectMeta("test"),
		Spec: kargoapi.StageSpec{
			Shard: "east",
			Vars: []kargoapi.ExpressionVariable{{
				Name:  "imageRepo",
				Value: "public.ecr.aws/nginx/nginx",
			}},
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "my-warehouse",
				},
				Sources: kargoapi.FreightSources{
					Direct:           true,
					Stages:           []string{"uat"},
					RequiredSoakTime: &metav1.Duration{Duration: time.Hour},
				},
			}},
			PromotionTemplate: &kargoapi.PromotionTemplate{
				Spec: kargoapi.PromotionTemplateSpec{
					Vars: []kargoapi.ExpressionVariable{{
						Name:  "branch",
						Value: "stage/${{ ctx.stage }}",
					}},
					Steps: []kargoapi.PromotionStep{{
						Uses: "git-clone",
						As:   "clone",
						Retry: &kargoapi.PromotionStepRetry{
							Timeout: &metav1.Duration{
								Duration: 5 * time.Minute,
							},
							ErrorThreshold: 3,
						},
						Config: &apiextensionsv1.JSON{Raw: []byte(
							`{"checkout":[{"branch":"main","path":"./src"}],` +
								`"depth":1,` +
								`"repoURL":"https://github.com/example/repo.git"}`,
						)},
					}},
				},
			},
		},
		Status: kargoapi.StageStatus{
			ObservedGeneration: 7,
			Conditions: []metav1.Condition{{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 7,
				LastTransitionTime: testTime,
				Reason:             "Healthy",
				Message:            "Stage is healthy",
			}},
			Health: &kargoapi.Health{
				Status: kargoapi.HealthStateHealthy,
				Issues: []string{},
				Output: &apiextensionsv1.JSON{Raw: []byte(
					`[{"applicationStatus":{"health":{"status":"Healthy"}}}]`,
				)},
			},
			FreightHistory: kargoapi.FreightHistory{{
				ID: "8a3b5c1d9e7f",
				Freight: map[string]kargoapi.FreightReference{
					"Warehouse/my-warehouse": {
						Name: "f7bc4db5eb3a...",
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "my-warehouse",
						},
						Images: []kargoapi.Image{{
							RepoURL: "public.ecr.aws/nginx/nginx",
							Tag:     "1.27.0",
							Digest:  "sha256:1e6a0da8d3ff...",
						}},
					},
				},
				VerificationHistory: []kargoapi.VerificationInfo{{
					ID:        "c1d2e3f4",
					Phase:     kargoapi.VerificationPhaseSuccessful,
					StartTime: &testTime,
					AnalysisRun: &kargoapi.AnalysisRunReference{
						Namespace: "kargo-demo",
						Name:      "test.01j0...",
						Phase:     "Successful",
					},
				}},
			}},
			LastPromotion: &kargoapi.PromotionReference{
				Name:       "test.01j0abc.f7bc4db",
				FinishedAt: &testTime,
				Freight: &kargoapi.FreightReference{
					Name: "f7bc4db5eb3a...",
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "my-warehouse",
					},
				},
				Status: &kargoapi.PromotionStatus{
					Phase:   kargoapi.PromotionPhaseSucceeded,
					Message: "promotion succeeded",
				},
			},
		},
	}
	roundTrip(t, stage, &gen.Stage{})
}

func TestFreightRoundTrip(t *testing.T) {
	freight := &kargoapi.Freight{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Freight",
		},
		ObjectMeta: testObjectMeta("f7bc4db5eb3a..."),
		Alias:      "wonky-wombat",
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "my-warehouse",
		},
		DiscoveredAt: &testTime,
		Commits: []kargoapi.GitCommit{{
			RepoURL:   "https://github.com/example/repo.git",
			ID:        "b2477396b5a80b8a1a72...",
			Branch:    "main",
			Message:   "feat: add feature",
			Author:    "Dev Eloper <dev@example.com>",
			Committer: "Dev Eloper <dev@example.com>",
		}},
		Images: []kargoapi.Image{{
			RepoURL: "public.ecr.aws/nginx/nginx",
			Tag:     "1.27.0",
			Digest:  "sha256:1e6a0da8d3ff...",
		}},
		Charts: []kargoapi.Chart{{
			RepoURL: "https://charts.example.com",
			Name:    "my-chart",
			Version: "1.2.3",
		}},
		Status: kargoapi.FreightStatus{
			CurrentlyIn: map[string]kargoapi.CurrentStage{
				"test": {Since: &testTime},
			},
			VerifiedIn: map[string]kargoapi.VerifiedStage{
				"test": {VerifiedAt: &testTime},
			},
			ApprovedFor: map[string]kargoapi.ApprovedStage{
				"prod": {ApprovedAt: &testTime},
			},
		},
	}
	roundTrip(t, freight, &gen.Freight{})
}

func TestPromotionRoundTrip(t *testing.T) {
	promotion := &kargoapi.Promotion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Promotion",
		},
		ObjectMeta: testObjectMeta("test.01j0abc.f7bc4db"),
		Spec: kargoapi.PromotionSpec{
			Stage:   "test",
			Freight: "f7bc4db5eb3a...",
			Vars: []kargoapi.ExpressionVariable{{
				Name:  "branch",
				Value: "stage/test",
			}},
			Steps: []kargoapi.PromotionStep{{
				Uses: "http",
				As:   "notify",
				Retry: &kargoapi.PromotionStepRetry{
					Timeout:        &metav1.Duration{Duration: time.Minute},
					ErrorThreshold: 2,
				},
				Config: &apiextensionsv1.JSON{Raw: []byte(
					`{"headers":[{"name":"content-type","value":"application/json"}],` +
						`"method":"POST",` +
						`"url":"https://hooks.example.com/notify"}`,
				)},
			}},
		},
		Status: kargoapi.PromotionStatus{
			Phase:       kargoapi.PromotionPhaseSucceeded,
			Message:     "promotion succeeded",
			StartedAt:   &testTime,
			FinishedAt:  &testTime,
			CurrentStep: 0,
			Freight: &kargoapi.FreightReference{
				Name: "f7bc4db5eb3a...",
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "my-warehouse",
				},
			},
			State: &apiextensionsv1.JSON{Raw: []byte(
				`{"notify":{"response":{"status":"ok"}}}`,
			)},
		},
	}
	roundTrip(t, promotion, &gen.Promotion{})
}

func TestWarehouseRoundTrip(t *testing.T) {
	warehouse := &kargoapi.Warehouse{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Warehouse",
		},
		ObjectMeta: testObjectMeta("my-warehouse"),
		Spec: kargoapi.WarehouseSpec{
			Shard:                 "east",
			Interval:              metav1.Duration{Duration: 5 * time.Minute},
			FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
			// InternalSubscriptions, not Subscriptions, is what WarehouseSpec's
			// custom MarshalJSON serializes into the wire-format
			// "subscriptions" array.
			InternalSubscriptions: []kargoapi.RepoSubscription{{
				Image: &kargoapi.ImageSubscription{
					RepoURL:    "public.ecr.aws/nginx/nginx",
					Constraint: "^1.27.0",
				},
			}},
		},
		Status: kargoapi.WarehouseStatus{
			ObservedGeneration: 7,
			LastFreightID:      "f7bc4db5eb3a...",
			LastHandledRefresh: "2026-07-16T12:30:45Z",
			Conditions: []metav1.Condition{{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 7,
				LastTransitionTime: testTime,
				Reason:             "Synced",
				Message:            "Warehouse is synced",
			}},
		},
	}
	roundTrip(t, warehouse, &gen.Warehouse{})
}

func TestConfigMapRoundTrip(t *testing.T) {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: testObjectMeta("my-config"),
		Data: map[string]string{
			"region": "us-east-1",
			"env":    "test",
		},
	}
	roundTrip(t, configMap, &gen.V1ConfigMap{})
}

func TestSecretRoundTrip(t *testing.T) {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: testObjectMeta("my-credentials"),
		Type:       corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"username": []byte("kargo"),
			"password": []byte("hunter2"),
		},
	}
	roundTrip(t, secret, &gen.V1Secret{})
}

// --- Formerly-broken wire formats -------------------------------------------
//
// swagger.json used to model Quantity, IntOrString, MicroTime, and
// arbitrary-JSON (apiextensions.JSON) values structurally (swag reflected the
// Go structs, ignoring their custom MarshalJSON), so no generated client
// could round-trip their real wire formats. That was fixed in the canonical
// spec (.swaggo overrides + fix-swagger-spec.sh Pass 4); these tests pin the
// corrected wire formats.

// requireLossless asserts the model faithfully round-trips the wire JSON.
func requireLossless(t *testing.T, wire string, model any) {
	t.Helper()
	require.NoError(t, json.Unmarshal([]byte(wire), model))
	replay, err := json.Marshal(model)
	require.NoError(t, err)
	require.JSONEq(t, wire, string(replay))
}

func TestWireFormat_IntOrString(t *testing.T) {
	// IntOrString marshals as a bare scalar: integer or string.
	wire := `{"name":"success-rate","count":5,"failureLimit":"20%"}`
	requireLossless(t, wire, &gen.RolloutsMetric{})
}

func TestWireFormat_Quantity(t *testing.T) {
	// Quantity marshals as a string.
	wire := `{"limits":{"cpu":"100m","memory":"128Mi"}}`
	requireLossless(t, wire, &gen.V1ResourceRequirements{})
}

func TestWireFormat_MicroTime(t *testing.T) {
	// metav1.MicroTime marshals as an RFC3339 string with microseconds.
	wire := `{"eventTime":"2026-07-16T12:30:45.123456Z","reason":"Promoted"}`
	requireLossless(t, wire, &gen.V1Event{})
}
