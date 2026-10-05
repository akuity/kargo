package server

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

func Test_server_authorize(t *testing.T) {
	t.Parallel()
	errDenied := errors.New("denied")
	s := &server{
		authorizeFn: func(
			_ context.Context,
			verb string,
			gvr schema.GroupVersionResource,
			subresource string,
			key client.ObjectKey,
		) error {
			require.Equal(t, "promote", verb)
			require.Equal(t, kargoapi.GroupVersion.WithResource("stages"), gvr)
			require.Empty(t, subresource)
			require.Equal(t, client.ObjectKey{Namespace: "kargo-demo", Name: "uat"}, key)
			return errDenied
		},
	}
	err := s.authorize(t.Context(), can.Promote().Stage("kargo-demo", "uat"))
	require.ErrorIs(t, err, errDenied)
}
