package option

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

type getVersionInfoHandlerFunc func(
	context.Context,
	*connect.Request[svcv1alpha1.GetVersionInfoRequest],
) (*connect.Response[svcv1alpha1.GetVersionInfoResponse], error)

type testErrorInterceptorServer struct {
	svcv1alpha1connect.UnimplementedKargoServiceHandler
	getVersionInfoHandler getVersionInfoHandlerFunc
}

func newTestErrorInterceptorServer(
	getVersionInfoHandler getVersionInfoHandlerFunc,
) *testErrorInterceptorServer {
	return &testErrorInterceptorServer{
		getVersionInfoHandler: getVersionInfoHandler,
	}
}

func (s *testErrorInterceptorServer) GetVersionInfo(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetVersionInfoRequest],
) (*connect.Response[svcv1alpha1.GetVersionInfoResponse], error) {
	return s.getVersionInfoHandler(ctx, req)
}

func TestErrorInterceptor(t *testing.T) {
	testCases := map[string]struct {
		handlerFunc        getVersionInfoHandlerFunc
		errExpected        bool
		expectedStatusCode connect.Code
	}{
		"no error": {
			handlerFunc: func(
				context.Context,
				*connect.Request[svcv1alpha1.GetVersionInfoRequest],
			) (*connect.Response[svcv1alpha1.GetVersionInfoResponse], error) {
				return connect.NewResponse(&svcv1alpha1.GetVersionInfoResponse{}), nil
			},
			errExpected: false,
		},
		"interceptor should not unwrap connect error with explicit status code": {
			handlerFunc: func(
				context.Context,
				*connect.Request[svcv1alpha1.GetVersionInfoRequest],
			) (*connect.Response[svcv1alpha1.GetVersionInfoResponse], error) {
				return nil, connect.NewError(
					connect.CodeInternal,
					kubeerr.NewForbidden(schema.GroupResource{}, "", nil),
				)
			},
			errExpected:        true,
			expectedStatusCode: connect.CodeInternal,
		},
		"interceptor should not unwrap connect error with unknown status code": {
			handlerFunc: func(
				context.Context,
				*connect.Request[svcv1alpha1.GetVersionInfoRequest],
			) (*connect.Response[svcv1alpha1.GetVersionInfoResponse], error) {
				return nil, connect.NewError(
					connect.CodeUnknown,
					kubeerr.NewForbidden(schema.GroupResource{}, "", nil),
				)
			},
			errExpected:        true,
			expectedStatusCode: connect.CodeUnknown,
		},
		"interceptor should wrap error with appropriate status code if possible": {
			handlerFunc: func(
				context.Context,
				*connect.Request[svcv1alpha1.GetVersionInfoRequest],
			) (*connect.Response[svcv1alpha1.GetVersionInfoResponse], error) {
				return nil, kubeerr.NewForbidden(schema.GroupResource{}, "", nil)
			},
			errExpected:        true,
			expectedStatusCode: connect.CodePermissionDenied,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			opt := connect.WithInterceptors(newErrorInterceptor())
			mux := http.NewServeMux()
			handler := newTestErrorInterceptorServer(tc.handlerFunc)
			mux.Handle(svcv1alpha1connect.NewKargoServiceHandler(handler, opt))
			srv := httptest.NewServer(mux)
			srv.EnableHTTP2 = true
			t.Cleanup(srv.Close)

			cli := svcv1alpha1connect.NewKargoServiceClient(srv.Client(), srv.URL, connect.WithGRPC())
			_, err := cli.GetVersionInfo(
				context.Background(),
				connect.NewRequest(&svcv1alpha1.GetVersionInfoRequest{}),
			)
			if tc.errExpected {
				require.Error(t, err)
				require.Equal(t, tc.expectedStatusCode, connect.CodeOf(err))
			} else {
				require.Nil(t, err)
			}
		})
	}
}
