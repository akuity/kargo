package get

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	pkgerrors "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

type filterProjectsFunc func(names ...string) ([]runtime.Object, error)

func filterProjects(
	ctx context.Context,
	kargoSvcCli svcv1alpha1connect.KargoServiceClient,
) (filterProjectsFunc, error) {
	resp, err := kargoSvcCli.ListProjects(ctx, connect.NewRequest(&v1alpha1.ListProjectsRequest{
		/* explicitly empty */
	}))
	if err != nil {
		return nil, pkgerrors.Wrap(err, "list projects")
	}
	return func(names ...string) ([]runtime.Object, error) {
		res := make([]runtime.Object, 0, len(resp.Msg.GetProjects()))
		if len(names) == 0 {
			for _, p := range resp.Msg.GetProjects() {
				res = append(res, typesv1alpha1.FromProjectProto(p))
			}
			return res, nil
		}

		var resErr error
		projects := make(map[string]*unstructured.Unstructured, len(resp.Msg.GetProjects()))
		for _, p := range resp.Msg.GetProjects() {
			projects[p.GetName()] = typesv1alpha1.FromProjectProto(p)
		}
		for _, name := range names {
			if project, ok := projects[name]; ok {
				res = append(res, project)
			} else {
				resErr = errors.Join(err, pkgerrors.Errorf("project %q not found", name))
			}
		}
		return res, resErr
	}, nil
}
