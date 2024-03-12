package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteProject(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteProjectRequest],
) (*connect.Response[svcv1alpha1.DeleteProjectResponse], error) {
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("name should not be empty"),
		)
	}
	if err := s.client.Delete(
		ctx,
		&kargoapi.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(
				connect.CodeNotFound,
				fmt.Errorf("project %q not found", name),
			)
		}
		return nil, fmt.Errorf("delete project: %w", err)
	}
	return connect.NewResponse(&svcv1alpha1.DeleteProjectResponse{}), nil
}
