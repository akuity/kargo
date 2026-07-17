// Package generatedv2test proves (or disproves) wire compatibility between
// the openapi-generator-generated Go client (pkg/client/generatedv2) and the
// canonical serialization produced by the real Kargo/Kubernetes API types.
//
// Method: construct richly-populated resources using the canonical types
// (github.com/akuity/kargo/api/v1alpha1 and k8s.io/api), marshal them with
// encoding/json -- byte-for-byte what the Kargo API server sends -- then
// unmarshal into the generated model and re-marshal. Round-trip output must
// be JSON-equal to ground truth.
//
// Also exercises three known, PRE-EXISTING spec-infidelity defects (swag, in
// producing swagger.json, reflects Go structs and ignores their custom
// MarshalJSON) that the CURRENT go-swagger client (pkg/client/generated)
// still has, proving the new client fixes them and the old one still
// doesn't (i.e. strict improvement, not a side-by-side regression):
//
//   - Quantity:    spec says object{Format}; wire format is a string ("100m")
//   - IntOrString: spec says object{intVal,strVal,type}; wire is int-or-string
//   - V1MicroTime: spec says object{"time.Time"}; wire format is a string
package generatedv2test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	oldmodels "github.com/akuity/kargo/pkg/client/generated/models"
	gen "github.com/akuity/kargo/pkg/client/generatedv2"
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
		},
		Status: kargoapi.StageStatus{
			Conditions: []metav1.Condition{{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 7,
				LastTransitionTime: testTime,
				Reason:             "Steady",
				Message:            "Stage is healthy and up to date",
			}},
			Health: &kargoapi.Health{
				Status: kargoapi.HealthStateHealthy,
				Output: &apiextensionsv1.JSON{
					Raw: []byte(`[{"check":"ok"}]`),
				},
			},
			FreightHistory: kargoapi.FreightHistory{{
				ID: "collection-1",
				Freight: map[string]kargoapi.FreightReference{
					"my-warehouse": {Name: "abc123"},
				},
				VerificationHistory: kargoapi.VerificationInfoStack{{
					ID:        "verification-1",
					Actor:     "kargo-controller",
					StartTime: &testTime,
					Phase:     kargoapi.VerificationPhaseSuccessful,
					Message:   "verification succeeded",
					AnalysisRun: &kargoapi.AnalysisRunReference{
						Namespace: "kargo-demo",
						Name:      "test-analysis-run",
						Phase:     "Successful",
					},
					FinishTime: &testTime,
				}},
			}},
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
		ObjectMeta: testObjectMeta("abc123"),
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "my-warehouse",
		},
		Commits: []kargoapi.GitCommit{{
			RepoURL: "https://github.com/example/repo.git",
			ID:      "abc123",
			Branch:  "main",
		}},
		Images: []kargoapi.Image{{
			RepoURL: "public.ecr.aws/nginx/nginx",
			Tag:     "1.27.0",
			Digest:  "sha256:abc123",
		}},
		Status: kargoapi.FreightStatus{
			ApprovedFor: map[string]kargoapi.ApprovedStage{
				"uat": {ApprovedAt: &testTime},
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
		ObjectMeta: testObjectMeta("test-promo"),
		Spec: kargoapi.PromotionSpec{
			Stage:   "uat",
			Freight: "abc123",
			Steps: []kargoapi.PromotionStep{{
				Uses: "git-clone",
				Config: &apiextensionsv1.JSON{
					Raw: []byte(`{"repoURL":"https://github.com/example/repo.git"}`),
				},
			}},
		},
		Status: kargoapi.PromotionStatus{
			Phase: kargoapi.PromotionPhaseSucceeded,
			State: &apiextensionsv1.JSON{
				Raw: []byte(`{"gitCloneOutput":{"commit":"abc123"}}`),
			},
			FinishedAt: &testTime,
		},
	}
	roundTrip(t, promotion, &gen.Promotion{})
}

func TestConfigMapRoundTrip(t *testing.T) {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: testObjectMeta("test-cm"),
		Data: map[string]string{
			"key": "value",
		},
		BinaryData: map[string][]byte{
			"bin": {0x01, 0x02, 0x03},
		},
	}
	roundTrip(t, cm, &gen.V1ConfigMap{})
}

func TestSecretRoundTrip(t *testing.T) {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: testObjectMeta("test-secret"),
		Type:       corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"username": []byte("kargo"),
			"password": []byte("hunter2"),
		},
	}
	roundTrip(t, secret, &gen.V1Secret{})
}

// --- Pre-existing spec-infidelity defects -----------------------------------

// requireLossless asserts the model faithfully round-trips the wire JSON.
func requireLossless(t *testing.T, wire string, model any) {
	t.Helper()
	require.NoError(t, json.Unmarshal([]byte(wire), model))
	replay, err := json.Marshal(model)
	require.NoError(t, err)
	require.JSONEq(t, wire, string(replay))
}

// requireDefective asserts the model CANNOT faithfully round-trip the wire
// JSON: it either fails to unmarshal it or re-marshals it to something
// different.
func requireDefective(t *testing.T, wire string, model any) {
	t.Helper()
	if err := json.Unmarshal([]byte(wire), model); err != nil {
		t.Logf("old client: unmarshal fails: %v", err)
		return
	}
	replay, err := json.Marshal(model)
	require.NoError(t, err)
	require.False(
		t, jsonEqual(t, wire, string(replay)),
		"old client round-tripped losslessly; defect is FIXED there -- "+
			"update this test",
	)
	t.Logf(
		"old client: silent data corruption:\n  wire:   %s\n  replay: %s",
		wire, replay,
	)
}

func jsonEqual(t *testing.T, a, b string) bool {
	t.Helper()
	var av, bv any
	require.NoError(t, json.Unmarshal([]byte(a), &av))
	require.NoError(t, json.Unmarshal([]byte(b), &bv))
	aNorm, err := json.Marshal(av)
	require.NoError(t, err)
	bNorm, err := json.Marshal(bv)
	require.NoError(t, err)
	return string(aNorm) == string(bNorm)
}

func TestSpecDefect_IntOrString(t *testing.T) {
	wire := `{"name":"success-rate","count":5,"failureLimit":"20%"}`
	requireLossless(t, wire, &gen.RolloutsMetric{})
	requireDefective(t, wire, &oldmodels.RolloutsMetric{})
}

func TestSpecDefect_Quantity(t *testing.T) {
	wire := `{"limits":{"cpu":"100m","memory":"128Mi"}}`
	requireLossless(t, wire, &gen.V1ResourceRequirements{})
	requireDefective(t, wire, &oldmodels.V1ResourceRequirements{})
}

func TestSpecDefect_MicroTime(t *testing.T) {
	wire := `{"eventTime":"2026-07-16T12:30:45.123456Z","reason":"Promoted"}`
	requireLossless(t, wire, &gen.V1Event{})
	requireDefective(t, wire, &oldmodels.V1Event{})
}
