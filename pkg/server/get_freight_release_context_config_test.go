package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/user"
)

func Test_server_getFreightReleaseContextConfig(t *testing.T) {
	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "example"}}
	freight := &kargoapi.Freight{ObjectMeta: metav1.ObjectMeta{
		Name: "release", Namespace: project.Name,
		Labels: map[string]string{kargoapi.LabelKeyAlias: "release-alias"},
	}}
	clusterMapping := &kargoapi.ReleaseContextConfig{ImageAnnotations: kargoapi.ImageAnnotationMappings{
		CommitSubject: "com.example.subject",
		CommitAuthor:  "com.example.author",
	}}
	projectMapping := &kargoapi.ReleaseContextConfig{ImageAnnotations: kargoapi.ImageAnnotationMappings{
		CommitSubject: "dev.example.summary",
	}}
	clusterConfig := &kargoapi.ClusterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec:       kargoapi.ClusterConfigSpec{ReleaseContext: clusterMapping},
	}
	projectConfig := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{Name: project.Name, Namespace: project.Name},
		Spec:       kargoapi.ProjectConfigSpec{ReleaseContext: projectMapping},
	}
	emptyProjectConfig := projectConfig.DeepCopy()
	emptyProjectConfig.Spec.ReleaseContext = &kargoapi.ReleaseContextConfig{}
	inheritingProjectConfig := projectConfig.DeepCopy()
	inheritingProjectConfig.Spec.ReleaseContext = nil
	otherProjectConfig := projectConfig.DeepCopy()
	otherProjectConfig.Name = "other"
	otherProjectConfig.Namespace = "other"

	testCases := []struct {
		name       string
		objects    []client.Object
		freight    string
		failKind   string
		wantStatus int
		want       kargoapi.ReleaseContextConfig
	}{
		{name: "missing project", wantStatus: http.StatusNotFound},
		{name: "missing freight", objects: []client.Object{project}, wantStatus: http.StatusNotFound},
		{name: "no configuration", objects: []client.Object{project, freight}, wantStatus: http.StatusOK},
		{
			name: "cluster defaults", objects: []client.Object{project, freight, clusterConfig},
			wantStatus: http.StatusOK, want: *clusterMapping,
		},
		{
			name: "project only", objects: []client.Object{project, freight, projectConfig},
			wantStatus: http.StatusOK, want: *projectMapping,
		},
		{
			name:       "project replaces the complete cluster mapping",
			objects:    []client.Object{project, freight, clusterConfig, projectConfig},
			wantStatus: http.StatusOK, want: *projectMapping,
		},
		{
			name:       "empty project setting disables custom interpretation",
			objects:    []client.Object{project, freight, clusterConfig, emptyProjectConfig},
			wantStatus: http.StatusOK,
		},
		{
			name:       "omitted project setting inherits defaults",
			objects:    []client.Object{project, freight, clusterConfig, inheritingProjectConfig},
			wantStatus: http.StatusOK, want: *clusterMapping,
		},
		{
			name:       "other projects do not affect the mapping",
			objects:    []client.Object{project, freight, clusterConfig, otherProjectConfig},
			wantStatus: http.StatusOK, want: *clusterMapping,
		},
		{
			name: "freight alias", objects: []client.Object{project, freight, clusterConfig},
			freight: "release-alias", wantStatus: http.StatusOK, want: *clusterMapping,
		},
		{
			name:     "project read failure is not treated as absent configuration",
			objects:  []client.Object{project, freight, clusterConfig},
			failKind: "ProjectConfig", wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "cluster read failure is surfaced",
			objects:  []client.Object{project, freight},
			failKind: "ClusterConfig", wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "project override does not depend on a cluster read",
			objects:  []client.Object{project, freight, projectConfig},
			failKind: "ClusterConfig", wantStatus: http.StatusOK, want: *projectMapping,
		},
	}

	for _, testCase := range testCases {
		nameOrAlias := testCase.freight
		if nameOrAlias == "" {
			nameOrAlias = freight.Name
		}
		builder := fake.NewClientBuilder().WithObjects(testCase.objects...).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(
				ctx context.Context,
				c client.WithWatch,
				key client.ObjectKey,
				obj client.Object,
				opts ...client.GetOption,
			) error {
				if testCase.failKind == "ClusterConfig" {
					if _, ok := obj.(*kargoapi.ClusterConfig); ok {
						return errors.New("cluster configuration unavailable")
					}
				}
				if testCase.failKind == "ProjectConfig" {
					if _, ok := obj.(*kargoapi.ProjectConfig); ok {
						return errors.New("project configuration unavailable")
					}
				}
				return c.Get(ctx, key, obj, opts...)
			},
		})
		testRESTEndpoint(t, nil, http.MethodGet,
			"/v1beta1/projects/example/freight/"+nameOrAlias+"/release-context-config",
			[]restTestCase{{
				name: testCase.name, clientBuilder: builder,
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, testCase.wantStatus, w.Code, w.Body.String())
					if testCase.wantStatus == http.StatusOK {
						var result kargoapi.ReleaseContextConfig
						require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
						require.Equal(t, testCase.want, result)
					}
				},
			}},
		)
	}
}

func Test_server_getFreightReleaseContextConfig_authorization(t *testing.T) {
	for _, allowFreight := range []bool{false, true} {
		configReads := 0
		var authorizedResources []string
		testRESTEndpoint(t, nil, http.MethodGet,
			"/v1beta1/projects/example/freight/release/release-context-config",
			[]restTestCase{{
				name: map[bool]string{false: "denied freight read", true: "freight reader without config access"}[allowFreight],
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "example"}},
					&kargoapi.Freight{ObjectMeta: metav1.ObjectMeta{Name: "release", Namespace: "example"}},
				).WithInterceptorFuncs(interceptor.Funcs{
					Create: func(
						_ context.Context,
						_ client.WithWatch,
						obj client.Object,
						_ ...client.CreateOption,
					) error {
						review, ok := obj.(*authv1.SubjectAccessReview)
						require.True(t, ok)
						attrs := review.Spec.ResourceAttributes
						authorizedResources = append(authorizedResources, attrs.Resource)
						require.Equal(t, "get", attrs.Verb)
						require.Equal(t, "example", attrs.Namespace)
						require.Equal(t, "release", attrs.Name)
						review.Status.Allowed = allowFreight && attrs.Resource == "freights"
						return nil
					},
					Get: func(
						ctx context.Context,
						c client.WithWatch,
						key client.ObjectKey,
						obj client.Object,
						opts ...client.GetOption,
					) error {
						switch obj.(type) {
						case *kargoapi.ClusterConfig, *kargoapi.ProjectConfig:
							configReads++
						}
						return c.Get(ctx, key, obj, opts...)
					},
				}),
				serverSetup: func(t *testing.T, s *server) {
					internalClient := s.client.InternalClient()
					var err error
					s.client, err = kubernetes.NewClient(t.Context(), &rest.Config{}, kubernetes.ClientOptions{
						NewInternalClient: func(
							context.Context, *rest.Config, *runtime.Scheme, string,
						) (client.WithWatch, error) {
							return internalClient, nil
						},
					})
					require.NoError(t, err)
				},
				ctxSetup: func(ctx context.Context) context.Context {
					return user.ContextWithInfo(ctx, user.Info{
						KubernetesUserInfo: &authnv1.UserInfo{Username: "reader"},
					})
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, []string{"freights"}, authorizedResources)
					if allowFreight {
						require.Equal(t, http.StatusOK, w.Code, w.Body.String())
						require.Equal(t, 2, configReads)
					} else {
						require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
						require.Zero(t, configReads)
					}
				},
			}},
		)
	}
}
