package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteStageRequest],
) (*connect.Response[svcv1alpha1.DeleteStageResponse], error) {
	if err := validateProjectAndStageNonEmpty(req.Msg.GetProject(), req.Msg.GetName()); err != nil {
		return nil, err
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	var stage kargoapi.Stage
	key := client.ObjectKey{
		Namespace: req.Msg.GetProject(),
		Name:      req.Msg.GetName(),
	}
	if err := s.client.Get(ctx, key, &stage); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("stage %q not found", key.String()))
		}
		return nil, connect.NewError(getCodeFromError(err), errors.Wrap(err, "get stage"))
	}

	approvedFreightList := kargoapi.FreightList{}
	err := s.listFreightFn(
		ctx,
		&approvedFreightList,
		&client.ListOptions{
			Namespace: req.Msg.GetProject(),
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.FreightApprovedForStagesIndexField,
				stage.Name,
			),
		},
	)

	if err != nil {
		return nil, connect.NewError(getCodeFromError(err), errors.Wrap(err, "list freight"))
	}

	for i := range approvedFreightList.Items {
		freight := approvedFreightList.Items[i]

		newStatus := *freight.Status.DeepCopy()
		if newStatus.ApprovedFor == nil {
			newStatus.ApprovedFor = map[string]kargoapi.ApprovedStage{}
		}

		delete(newStatus.ApprovedFor, stage.Name)

		if err := kubeclient.PatchStatus(
			ctx,
			s.client,
			&freight,
			func(status *kargoapi.FreightStatus) {
				*status = newStatus
			},
		); err != nil {
			return nil, connect.NewError(getCodeFromError(err), errors.Wrap(err, "patch status"))
		}
	}

	if err := s.client.Delete(ctx, &stage); err != nil && !kubeerr.IsNotFound(err) {
		return nil, connect.NewError(getCodeFromError(err), errors.Wrap(err, "delete stage"))
	}
	return connect.NewResponse(&svcv1alpha1.DeleteStageResponse{}), nil
}
