package get

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	pkgerrors "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

type filterStagesFunc func(names ...string) ([]runtime.Object, error)

func filterStages(
	ctx context.Context,
	kargoSvcCli svcv1alpha1connect.KargoServiceClient,
	project string,
) (filterStagesFunc, error) {
	resp, err := kargoSvcCli.ListStages(ctx, connect.NewRequest(&v1alpha1.ListStagesRequest{
		Project: project,
	}))
	if err != nil {
		return nil, pkgerrors.Wrap(err, "list stages")
	}
	return func(names ...string) ([]runtime.Object, error) {
		res := make([]runtime.Object, 0, len(resp.Msg.GetStages()))
		if len(names) == 0 {
			for _, s := range resp.Msg.GetStages() {
				res = append(res, typesv1alpha1.FromStageProto(s))
			}
			return res, nil
		}

		var resErr error
		stages := make(map[string]*kargoapi.Stage, len(resp.Msg.GetStages()))
		for _, s := range resp.Msg.GetStages() {
			stages[s.GetMetadata().GetName()] = typesv1alpha1.FromStageProto(s)
		}
		for _, name := range names {
			if stage, ok := stages[name]; ok {
				res = append(res, stage)
			} else {
				resErr = errors.Join(err, pkgerrors.Errorf("stage %q not found", name))
			}
		}
		return res, resErr
	}, nil
}
