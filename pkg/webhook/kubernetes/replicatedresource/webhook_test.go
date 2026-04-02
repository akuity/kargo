package replicatedresource

import (
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	authnv1 "k8s.io/api/authentication/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	libWebhook "github.com/akuity/kargo/pkg/webhook/kubernetes"
)

const testManagementControllerUsername = "system:serviceaccount:kargo:kargo-management-controller"

func TestNewWebhook(t *testing.T) {
	w := newWebhook(libWebhook.Config{
		ManagementControllerUsername: testManagementControllerUsername,
	})
	require.Equal(
		t,
		testManagementControllerUsername,
		w.cfg.ManagementControllerUsername,
	)
}

func TestHandle(t *testing.T) {
	cfg := libWebhook.Config{
		ManagementControllerUsername: testManagementControllerUsername,
	}
	testCases := []struct {
		name   string
		req    admission.Request
		assert func(*testing.T, admission.Response)
	}{
		{
			name: "management controller CREATE is allowed",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					UserInfo: authnv1.UserInfo{
						Username: testManagementControllerUsername,
					},
				},
			},
			assert: func(t *testing.T, resp admission.Response) {
				require.True(t, resp.Allowed)
			},
		},
		{
			name: "management controller UPDATE is allowed",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					UserInfo: authnv1.UserInfo{
						Username: testManagementControllerUsername,
					},
				},
			},
			assert: func(t *testing.T, resp admission.Response) {
				require.True(t, resp.Allowed)
			},
		},
		{
			name: "management controller DELETE is allowed",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					UserInfo: authnv1.UserInfo{
						Username: testManagementControllerUsername,
					},
				},
			},
			assert: func(t *testing.T, resp admission.Response) {
				require.True(t, resp.Allowed)
			},
		},
		{
			name: "other controlplane component is denied",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					UserInfo: authnv1.UserInfo{
						Username: "system:serviceaccount:kargo:kargo-api",
					},
				},
			},
			assert: func(t *testing.T, resp admission.Response) {
				require.False(t, resp.Allowed)
			},
		},
		{
			name: "non-controlplane user CREATE is denied",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					UserInfo: authnv1.UserInfo{
						Username: "some-user",
					},
				},
			},
			assert: func(t *testing.T, resp admission.Response) {
				require.False(t, resp.Allowed)
			},
		},
		{
			name: "non-controlplane user UPDATE is denied",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					UserInfo: authnv1.UserInfo{
						Username: "some-user",
					},
				},
			},
			assert: func(t *testing.T, resp admission.Response) {
				require.False(t, resp.Allowed)
			},
		},
		{
			name: "non-controlplane user DELETE is denied",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					UserInfo: authnv1.UserInfo{
						Username: "some-user",
					},
				},
			},
			assert: func(t *testing.T, resp admission.Response) {
				require.False(t, resp.Allowed)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			w := newWebhook(cfg)
			resp := w.Handle(t.Context(), testCase.req)
			testCase.assert(t, resp)
		})
	}
}
