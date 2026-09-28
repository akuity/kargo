package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/user"
)

func Test_server_approveFreight(t *testing.T) {
	const testStageName = "fake-stage"
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-project",
		},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-freight",
			Namespace: testProject.Name,
		},
		Origin: testOrigin,
	}
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: testProject.Name,
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{Origin: testOrigin}},
		},
	}
	// Same name as testStage, but doesn't request Freight from testOrigin.
	testStageWithoutRequest := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: testProject.Name,
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/freight/"+testFreight.Name+"/approve?stage="+testStageName,
		[]restTestCase{
			{
				name:          "Project not found",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Freight not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Stage not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
				).WithStatusSubresource(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "not authorized to approve (not authorized to promote)",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight, testStage).
					WithStatusSubresource(testFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return apierrors.NewForbidden(
							kargoapi.GroupVersion.WithResource("stages").GroupResource(),
							testStageName,
							errors.New("not authorized"),
						)
					}
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name: "Stage does not request Freight from origin",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight, testStageWithoutRequest).
					WithStatusSubresource(testFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return nil
					}
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
					require.Contains(t, w.Body.String(), "does not request Freight from origin")

					// Verify the Freight was NOT approved for the Stage
					freight := &kargoapi.Freight{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testFreight),
						freight,
					)
					require.NoError(t, err)
					require.False(t, freight.IsApprovedFor(testStageName))
				},
			},
			{
				name: "approves Freight",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight, testStage).
					WithStatusSubresource(testFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return nil
					}
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify the Freight was approved for the Stage
					freight := &kargoapi.Freight{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testFreight),
						freight,
					)
					require.NoError(t, err)
					require.True(t, freight.IsApprovedFor(testStage.Name))
					require.Contains(t, freight.Status.ApprovedFor, testStage.Name)
				},
			},
		},
	)
}

// newPromoterClient builds a kubernetes.Client that enforces real
// authorization, unlike the harness's SkipAuthorization: true default. It
// answers SubjectAccessReviews the way the chart's kargo-promoter Role would:
// promote on the named Stage, and nothing else -- notably not patch on
// freights/status, which is no longer granted to any built-in role.
func newPromoterClient(
	t *testing.T,
	store *fake.ClientBuilder,
	stageName string,
) (kubernetes.Client, client.WithWatch) {
	t.Helper()
	scheme := newTestScheme(t)
	require.NoError(t, authv1.AddToScheme(scheme))
	internalClient := store.
		WithScheme(scheme).
		WithRESTMapper(testRESTMapper(scheme)).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(
				ctx context.Context,
				c client.WithWatch,
				obj client.Object,
				opts ...client.CreateOption,
			) error {
				review, ok := obj.(*authv1.SubjectAccessReview)
				if !ok {
					return c.Create(ctx, obj, opts...)
				}
				ra := review.Spec.ResourceAttributes
				switch {
				case ra.Group == kargoapi.GroupVersion.Group &&
					ra.Resource == "stages" && ra.Verb == "promote" &&
					ra.Name == stageName:
					review.Status.Allowed = true
				case ra.Group == kargoapi.GroupVersion.Group &&
					(ra.Resource == "freights" || ra.Resource == "stages") &&
					(ra.Verb == "get" || ra.Verb == "list" || ra.Verb == "watch"):
					// Mirrors the kargo-promoter Role's read access to these
					// resource types.
					review.Status.Allowed = true
				}
				// Anything else -- notably patch on freights/status -- is
				// intentionally left denied.
				return nil
			},
		}).
		Build()
	c, err := kubernetes.NewClient(
		t.Context(),
		&rest.Config{},
		kubernetes.ClientOptions{
			NewInternalClient: func(
				context.Context,
				*rest.Config,
				*runtime.Scheme,
				string,
			) (client.WithWatch, error) {
				return internalClient, nil
			},
		},
	)
	require.NoError(t, err)
	return c, internalClient
}

// Test_server_approveFreight_noFreightsStatusPatchRequired is a regression
// test for GHSA-rx5g-3338-f2mf. Before the fix, approving Freight required
// the caller to hold Kubernetes RBAC permission to patch freights/status,
// which -- since RBAC can't scope patch to a single status field -- also let
// a holder write status.verifiedIn directly, bypassing verification and soak
// requirements. Freight approval now writes status through the internal
// client, so a caller who holds only the promote verb on the target Stage
// can approve Freight without any RBAC grant on freights/status.
func Test_server_approveFreight_noFreightsStatusPatchRequired(t *testing.T) {
	const testStageName = "fake-stage"
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-freight",
			Namespace: testProject.Name,
		},
		Origin: testOrigin,
	}
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testStageName,
			Namespace: testProject.Name,
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{Origin: testOrigin}},
		},
	}
	// promoterUser mirrors the identity a real bearer token resolves to via
	// TokenReview when it belongs to a ServiceAccount bound only to the
	// kargo-promoter Role.
	promoterUser := user.Info{
		KubernetesUserInfo: &authnv1.UserInfo{
			Username: "system:serviceaccount:fake-project:promoter",
		},
	}

	// capturedClient is reassigned by serverSetup before assertions reads it.
	var capturedClient client.WithWatch

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost,
		"/v1beta1/projects/"+testProject.Name+"/freight/"+testFreight.Name+"/approve?stage="+testStageName,
		[]restTestCase{
			{
				name: "approves Freight without freights/status patch permission",
				serverSetup: func(t *testing.T, s *server) {
					s.client, capturedClient = newPromoterClient(
						t,
						fake.NewClientBuilder().WithObjects(testProject, testFreight, testStage).
							WithStatusSubresource(testFreight),
						testStageName,
					)
					s.authorizeFn = s.client.Authorize
					s.patchFreightStatusFn = s.patchFreightStatus
				},
				ctxSetup: func(ctx context.Context) context.Context {
					return user.ContextWithInfo(ctx, promoterUser)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					freight := &kargoapi.Freight{}
					require.NoError(t, capturedClient.Get(
						t.Context(),
						client.ObjectKeyFromObject(testFreight),
						freight,
					))
					require.True(t, freight.IsApprovedFor(testStageName))
				},
			},
		},
	)
}
