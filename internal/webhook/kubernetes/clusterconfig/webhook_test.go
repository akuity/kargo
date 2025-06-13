package clusterconfig

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
)

func Test_webhook_ValidateCreate(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        *kargoapi.ClusterConfig
		assertions func(*testing.T, admission.Warnings, error)
	}{
		{
			name: `invalid metadata: name is not "cluster"`,
			cfg: &kargoapi.ClusterConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "another-name",
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				require.Contains(
					t,
					statusErr.ErrStatus.Details.Causes[0].Message,
					`name "another-name" must be "cluster"`,
				)
				require.Equal(t, "metadata.name", statusErr.ErrStatus.Details.Causes[0].Field)

				require.Empty(t, warnings)
			},
		},
		{
			name: "duplicate webhook receiver names",
			cfg: &kargoapi.ClusterConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: api.ClusterConfigName,
				},
				Spec: kargoapi.ClusterConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{Name: "my-webhook-receiver"},
						{Name: "my-webhook-receiver"},
					},
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				require.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				require.Equal(t, 1, len(statusErr.ErrStatus.Details.Causes))

				require.Equal(
					t,
					"spec.webhookReceivers[1].name",
					statusErr.ErrStatus.Details.Causes[0].Field,
				)
				require.Equal(
					t,
					metav1.CauseTypeFieldValueInvalid,
					statusErr.ErrStatus.Details.Causes[0].Type,
				)
				require.Contains(
					t,
					statusErr.ErrStatus.Details.Causes[0].Message,
					"webhook receiver name already defined at spec.webhookReceivers[0]",
				)

				require.Empty(t, warnings)
			},
		},
		{
			name: "valid cluster config",
			cfg: &kargoapi.ClusterConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: api.ClusterConfigName,
				},
				Spec: kargoapi.ClusterConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{Name: "receiver-1"},
						{Name: "receiver-2"},
					},
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				require.NoError(t, err)
				require.Empty(t, warnings)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{}
			warnings, err := w.ValidateCreate(context.Background(), testCase.cfg)
			testCase.assertions(t, warnings, err)
		})
	}
}

func Test_webhook_ValidateUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        *kargoapi.ClusterConfig
		assertions func(*testing.T, admission.Warnings, error)
	}{
		{
			name: "duplicate webhook receiver names",
			cfg: &kargoapi.ClusterConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: api.ClusterConfigName,
				},
				Spec: kargoapi.ClusterConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{Name: "my-webhook-receiver"},
						{Name: "my-webhook-receiver"},
					},
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				require.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				require.Equal(t, 1, len(statusErr.ErrStatus.Details.Causes))

				require.Equal(
					t,
					"spec.webhookReceivers[1].name",
					statusErr.ErrStatus.Details.Causes[0].Field,
				)
				require.Equal(
					t,
					metav1.CauseTypeFieldValueInvalid,
					statusErr.ErrStatus.Details.Causes[0].Type,
				)
				require.Contains(
					t,
					statusErr.ErrStatus.Details.Causes[0].Message,
					"webhook receiver name already defined at spec.webhookReceivers[0]",
				)

				require.Empty(t, warnings)
			},
		},
		{
			name: "valid cluster config",
			cfg: &kargoapi.ClusterConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: api.ClusterConfigName,
				},
				Spec: kargoapi.ClusterConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{Name: "receiver-1"},
						{Name: "receiver-2"},
					},
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				require.NoError(t, err)
				require.Empty(t, warnings)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{}
			warnings, err := w.ValidateUpdate(context.Background(), nil, testCase.cfg)
			testCase.assertions(t, warnings, err)
		})
	}
}
