package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func TestUpdateFreightAlias(t *testing.T) {
	testCases := []struct {
		name       string
		req        *svcv1alpha1.UpdateFreightAliasRequest
		server     *server
		assertions func(*testing.T, error)
	}{
		{
			name:   "project not specified",
			req:    &svcv1alpha1.UpdateFreightAliasRequest{},
			server: &server{},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
			},
		},
		{
			name: "neither name nor existing alias specified",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project: "fake-project",
			},
			server: &server{},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
			},
		},
		{
			name: "new alias not specified",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project: "fake-project",
				Name:    "fake-freight",
			},
			server: &server{},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
			},
		},
		{
			name: "error validating project",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "error getting Freight",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "freight not found",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeNotFound, connErr.Code())
				require.Contains(t, connErr.Message(), "freight")
				require.Contains(t, connErr.Message(), "not found in namespace")
			},
		},
		{
			name: "error listing freight",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInternal, connErr.Code())
				require.Equal(t, "something went wrong", connErr.Message())
			},
		},
		{
			name: "alias is not unique",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight",
						},
					}, nil
				},
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "different-fake-freight",
							},
						},
					}
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeAlreadyExists, connErr.Code())
				require.Contains(
					t,
					connErr.Message(),
					"already used by another Freight resource",
				)
			},
		},
		{
			name: "error patching Freight",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				listFreightFn: func(
					_ context.Context,
					_ client.ObjectList,
					_ ...client.ListOption,
				) error {
					return nil
				},
				patchFreightAliasFn: func(
					context.Context,
					*kargoapi.Freight,
					string,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInternal, connErr.Code())
				require.Equal(t, "something went wrong", connErr.Message())
			},
		},
		{
			name: "success",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				listFreightFn: func(
					_ context.Context,
					_ client.ObjectList,
					_ ...client.ListOption,
				) error {
					return nil
				},
				patchFreightAliasFn: func(
					context.Context,
					*kargoapi.Freight,
					string,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.server.UpdateFreightAlias(
				context.Background(),
				connect.NewRequest(testCase.req),
			)
			testCase.assertions(t, err)
		})
	}
}

func Test_server_patchFreightAliasHandler(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	const (
		testOldAlias = "old-alias"
		testNewAlias = "new-alias"
	)
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-freight",
			Namespace: testProject.Name,
			Labels: map[string]string{
				kargoapi.LabelKeyAlias: testOldAlias,
			},
		},
		Alias: testOldAlias,
	}
	testBaseURL := "/v1beta1/projects/" + testProject.Name + "/freight/"
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPatch, testBaseURL+testFreight.Name+"/alias?newAlias="+testNewAlias,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight not found by name",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "new alias already in use by another piece of freight",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
					&kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other-freight",
							Namespace: testProject.Name,
							Labels:    map[string]string{kargoapi.LabelKeyAlias: testNewAlias},
						},
						Alias: testNewAlias,
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "updates alias by name",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify the Freight was updated in the cluster
					freight := &kargoapi.Freight{}
					err := c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject.Name, Name: testFreight.Name},
						freight,
					)
					require.NoError(t, err)
					require.Equal(t, testNewAlias, freight.Labels[kargoapi.LabelKeyAlias])
					require.Equal(t, testNewAlias, freight.Alias)
				},
			},
			{
				name: "updates alias by old alias",
				url:  testBaseURL + testOldAlias + "/alias?newAlias=" + testNewAlias,
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify the Freight was updated
					freight := &kargoapi.Freight{}
					err := c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject.Name, Name: testFreight.Name},
						freight,
					)
					require.NoError(t, err)
					require.Equal(t, testNewAlias, freight.Labels[kargoapi.LabelKeyAlias])
					require.Equal(t, testNewAlias, freight.Alias)
				},
			},
			{
				name: "newAlias query parameter is required",
				url:  testBaseURL + testFreight.Name + "/alias",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
		},
	)
}
