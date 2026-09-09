package webhook

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type fakeStageDefaulter struct {
	defaultFn func(ctx context.Context, stage *kargoapi.Stage) error
}

func (f fakeStageDefaulter) Default(ctx context.Context, stage *kargoapi.Stage) error {
	return f.defaultFn(ctx, stage)
}

func TestNewDefaultingWebhook(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	// A shorthand duration ("1h") that Default() never touches. It should
	// survive untouched, not get remarshaled to "1h0m0s".
	rawStage := []byte(`{
		"apiVersion": "kargo.akuity.io/v1alpha1",
		"kind": "Stage",
		"metadata": {"name": "test-stage", "namespace": "test-project"},
		"spec": {
			"promotionTemplate": {
				"spec": {
					"steps": [{"uses": "git-clone", "retry": {"timeout": "1h"}}]
				}
			}
		}
	}`)
	const untouchedDurationPath = "/spec/promotionTemplate/spec/steps/0/retry/timeout"

	testCases := []struct {
		name       string
		req        admission.Request
		defaultFn  func(ctx context.Context, stage *kargoapi.Stage) error
		assertions func(*testing.T, admission.Response)
	}{
		{
			name: "delete operation is always allowed without decoding",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
				},
			},
			defaultFn: func(context.Context, *kargoapi.Stage) error {
				t.Fatal("Default() should not be called for a Delete operation")
				return nil
			},
			assertions: func(t *testing.T, resp admission.Response) {
				require.True(t, resp.Allowed)
				require.Empty(t, resp.Patches)
			},
		},
		{
			name: "no-op defaulter produces no patch",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object:    runtime.RawExtension{Raw: rawStage},
				},
			},
			defaultFn: func(context.Context, *kargoapi.Stage) error {
				return nil
			},
			assertions: func(t *testing.T, resp admission.Response) {
				require.True(t, resp.Allowed)
				require.Empty(t, resp.Patches)
			},
		},
		{
			name: "genuine mutation is patched without disturbing untouched duration field",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object:    runtime.RawExtension{Raw: rawStage},
				},
			},
			defaultFn: func(_ context.Context, stage *kargoapi.Stage) error {
				stage.Labels = map[string]string{kargoapi.LabelKeyShard: "fake-shard"}
				return nil
			},
			assertions: func(t *testing.T, resp admission.Response) {
				require.True(t, resp.Allowed)

				var sawLabelPatch bool
				for _, p := range resp.Patches {
					require.NotEqualf(
						t, untouchedDurationPath, p.Path,
						"Default() did not touch this field, but it was patched to %v", p.Value,
					)
					if p.Path == "/metadata/labels" {
						sawLabelPatch = true
					}
				}
				require.True(t, sawLabelPatch, "expected a patch adding the shard label")
			},
		},
		{
			name: "plain defaulter error is denied",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object:    runtime.RawExtension{Raw: rawStage},
				},
			},
			defaultFn: func(context.Context, *kargoapi.Stage) error {
				return errors.New("something went wrong")
			},
			assertions: func(t *testing.T, resp admission.Response) {
				require.False(t, resp.Allowed)
				require.NotNil(t, resp.Result)
				require.Contains(t, resp.Result.Message, "something went wrong")
			},
		},
		{
			name: "apierrors-typed defaulter error carries its status",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object:    runtime.RawExtension{Raw: rawStage},
				},
			},
			defaultFn: func(context.Context, *kargoapi.Stage) error {
				return apierrors.NewBadRequest("invalid spec")
			},
			assertions: func(t *testing.T, resp admission.Response) {
				require.False(t, resp.Allowed)
				require.NotNil(t, resp.Result)
				require.Equal(t, metav1.StatusReasonBadRequest, resp.Result.Reason)
			},
		},
		{
			name: "malformed request body fails to decode",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object:    runtime.RawExtension{Raw: []byte("not json")},
				},
			},
			defaultFn: func(context.Context, *kargoapi.Stage) error {
				t.Fatal("Default() should not be called when decoding fails")
				return nil
			},
			assertions: func(t *testing.T, resp admission.Response) {
				require.False(t, resp.Allowed)
				require.NotNil(t, resp.Result)
				require.EqualValues(t, 400, resp.Result.Code)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			wh := NewDefaultingWebhook(
				scheme,
				&kargoapi.Stage{},
				fakeStageDefaulter{defaultFn: testCase.defaultFn},
			)
			ctx := admission.NewContextWithRequest(context.Background(), testCase.req)
			testCase.assertions(t, wh.Handle(ctx, testCase.req))
		})
	}
}
