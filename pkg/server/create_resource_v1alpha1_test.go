package server

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

// clusterScopedTestKinds mirrors the +kubebuilder:resource:scope=Cluster
// markers in api/v1alpha1. meta.NewDefaultRESTMapper can't infer scope from a
// scheme, so getting this wrong would make every kind look namespaced to fake
// clients and mask scope-dependent bugs in tests.
var clusterScopedTestKinds = map[string]bool{
	"Project":              true,
	"ClusterConfig":        true,
	"ClusterPromotionTask": true,
}

// testRESTMapper builds a RESTMapper from a scheme so fake clients can
// resolve a resource type's scope the way the production RESTMapper would.
func testRESTMapper(s *runtime.Scheme) meta.RESTMapper {
	m := meta.NewDefaultRESTMapper(nil)
	for gvk := range s.AllKnownTypes() {
		if strings.HasSuffix(gvk.Kind, "List") {
			continue
		}
		scope := meta.RESTScopeNamespace
		if clusterScopedTestKinds[gvk.Kind] {
			scope = meta.RESTScopeRoot
		}
		m.Add(gvk, scope)
	}
	return m
}

// newTestScheme builds the scheme used by tests that build their own
// kubernetes.Client.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))
	return testScheme
}

// authGatedClient is a hand-written test double for kubernetes.Client. This
// branch's real authorizing client submits a SubjectAccessReview through a
// brand-new client built from GetRestConfig, which needs a live cluster and
// can't be faked with a store interceptor. This double instead denies Create
// directly for the kinds named in deniedCreateKinds, simulating what a real
// SubjectAccessReview would deny for kargo-project-creator, while
// InternalClient() returns the same backing store unrestricted, matching
// real bypass behavior. Every other method (Get, Update, IsObjectNamespaced,
// etc.) is promoted straight from the embedded store.
type authGatedClient struct {
	client.WithWatch
	deniedCreateKinds map[string]bool
}

func (a *authGatedClient) Create(
	ctx context.Context,
	obj client.Object,
	opts ...client.CreateOption,
) error {
	if a.deniedCreateKinds[obj.GetObjectKind().GroupVersionKind().Kind] {
		gvk := obj.GetObjectKind().GroupVersionKind()
		return apierrors.NewForbidden(
			schema.GroupResource{Group: gvk.Group, Resource: strings.ToLower(gvk.Kind) + "s"},
			obj.GetName(),
			errors.New("create is not permitted"),
		)
	}
	return a.WithWatch.Create(ctx, obj, opts...)
}

func (a *authGatedClient) InternalClient() client.Client { return a.WithWatch }

func (a *authGatedClient) Authorize(
	context.Context, string, schema.GroupVersionResource, string, client.ObjectKey,
) error {
	return nil
}

func (a *authGatedClient) Watch(
	context.Context, client.Object, string, metav1.ListOptions,
) (watch.Interface, error) {
	panic("not implemented for this test double")
}

// newProjectCreatorClient builds an authGatedClient that denies Create for
// ClusterConfig/ClusterPromotionTask, mirroring the chart's
// kargo-project-creator ClusterRole (full access to Project, read-only on
// those two cluster-scoped kinds). Also returns the raw store so the caller
// can inspect it afterward.
func newProjectCreatorClient(
	t *testing.T,
	store *fake.ClientBuilder,
) (kubernetes.Client, client.WithWatch) {
	t.Helper()
	scheme := newTestScheme(t)
	internalClient := store.
		WithScheme(scheme).
		WithRESTMapper(testRESTMapper(scheme)).
		Build()
	return &authGatedClient{
		WithWatch: internalClient,
		deniedCreateKinds: map[string]bool{
			"ClusterConfig":        true,
			"ClusterPromotionTask": true,
		},
	}, internalClient
}

// mustManifest marshals objs to a multi-document YAML manifest, the shape the
// legacy Connect-RPC resource endpoints accept as raw bytes.
func mustManifest(t *testing.T, objs ...any) []byte {
	t.Helper()
	var buf bytes.Buffer
	for i, obj := range objs {
		if i > 0 {
			buf.WriteString("---\n")
		}
		b, err := sigyaml.Marshal(obj)
		require.NoError(t, err)
		buf.Write(b)
	}
	return buf.Bytes()
}

// TestCreateResource_clusterScopedNamespaceSpoofing is a regression test: a
// principal who could only create Projects could spoof a cluster-scoped
// resource's metadata.namespace to a just-created Project's name and get it
// created anyway, since Kubernetes ignores metadata.namespace for
// cluster-scoped kinds.
func TestCreateResource_clusterScopedNamespaceSpoofing(t *testing.T) {
	const projectName = "ghsa-repro-rpc"

	t.Run("control: direct create of a cluster-scoped resource is denied", func(t *testing.T) {
		s := &server{}
		s.client, _ = newProjectCreatorClient(t, fake.NewClientBuilder())

		_, err := s.CreateResource(context.Background(), connect.NewRequest(&svcv1alpha1.CreateResourceRequest{
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

			resp, err := s.CreateResource(context.Background(), connect.NewRequest(&svcv1alpha1.CreateResourceRequest{
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
				context.Background(),
				client.ObjectKey{Name: "exploit-task-rpc"},
				&kargoapi.ClusterPromotionTask{},
			)
			require.True(t, apierrors.IsNotFound(getErr), "ClusterPromotionTask must not have been persisted")
		},
	)
}
