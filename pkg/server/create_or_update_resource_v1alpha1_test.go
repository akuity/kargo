package server

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// TestCreateOrUpdateResource_clusterScopedNamespaceSpoofing is the legacy
// Connect-RPC CreateOrUpdateResource counterpart to
// Test_server_createResources_clusterScopedNamespaceSpoofing in
// create_resource_v1alpha1_test.go; see that test's doc comment for the full
// explanation. newProjectCreatorClient is defined there and reused here.
func TestCreateOrUpdateResource_clusterScopedNamespaceSpoofing(t *testing.T) {
	const projectName = "ghsa-repro-rpc"

	t.Run("control: direct upsert of a cluster-scoped resource is denied", func(t *testing.T) {
		s := &server{}
		s.client, _ = newProjectCreatorClient(t, fake.NewClientBuilder())

		_, err := s.CreateOrUpdateResource(t.Context(), connect.NewRequest(&svcv1alpha1.CreateOrUpdateResourceRequest{
			Manifest: mustManifest(t, &kargoapi.ClusterPromotionTask{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kargoapi.GroupVersion.String(),
					Kind:       "ClusterPromotionTask",
				},
				ObjectMeta: metav1.ObjectMeta{Name: "control-task-rpc"},
				Spec: kargoapi.PromotionTaskSpec{
					Steps: []kargoapi.PromotionStep{{Uses: "fail"}},
				},
			}),
		}))
		require.Error(t, err)
		require.True(t, apierrors.IsForbidden(err))
	})

	t.Run(
		"exploit: ClusterPromotionTask namespace spoofed to a just-created Project is still denied",
		func(t *testing.T) {
			s := &server{}
			var capturedClient client.WithWatch
			s.client, capturedClient = newProjectCreatorClient(t, fake.NewClientBuilder())

			resp, err := s.CreateOrUpdateResource(t.Context(), connect.NewRequest(&svcv1alpha1.CreateOrUpdateResourceRequest{
				Manifest: mustManifest(
					t,
					&kargoapi.Project{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "Project",
						},
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.ClusterPromotionTask{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "ClusterPromotionTask",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "exploit-task-rpc",
							Namespace: projectName,
						},
						Spec: kargoapi.PromotionTaskSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fail"}},
						},
					},
				),
			}))
			require.NoError(t, err)
			require.Len(t, resp.Msg.Results, 2)
			require.Empty(t, resp.Msg.Results[0].GetError(), "Project creation should succeed")
			require.NotEmpty(
				t, resp.Msg.Results[1].GetError(),
				"ClusterPromotionTask creation must be denied, not silently bypassed via a spoofed namespace",
			)
			require.Contains(t, resp.Msg.Results[1].GetError(), "forbidden")

			getErr := capturedClient.Get(
				t.Context(),
				client.ObjectKey{Name: "exploit-task-rpc"},
				&kargoapi.ClusterPromotionTask{},
			)
			require.True(t, apierrors.IsNotFound(getErr), "ClusterPromotionTask must not have been persisted")
		},
	)
}
