package api

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) CreateProject(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateProjectRequest],
) (*connect.Response[svcv1alpha1.CreateProjectResponse], error) {
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
	}
	project := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := s.client.Create(ctx, project); err != nil {
		return nil, errors.Wrapf(err, "error creating project %q", name)
	}
	return connect.NewResponse(&svcv1alpha1.CreateProjectResponse{
		Project: typesv1alpha1.ToProjectProto(*project),
	}), nil
}
