package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargov1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type WatchStageV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.WatchStageRequest],
	*connect.ServerStream[svcv1alpha1.WatchStageResponse],
) error

func WatchStageV1Alpha1(
	kubeCli client.Client,
	dynamicCli dynamic.Interface,
) WatchStageV1Alpha1Func {
	validateProject := newProjectValidator(kubeCli)
	stageCli := dynamicCli.Resource(kargov1alpha1.GroupVersion.WithResource("stages"))
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.WatchStageRequest],
		stream *connect.ServerStream[svcv1alpha1.WatchStageResponse],
	) error {
		if req.Msg.GetProject() == "" {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
		}
		if err := validateProject(ctx, req.Msg.GetProject()); err != nil {
			return err
		}

		if req.Msg.GetName() != "" {
			if err := kubeCli.Get(ctx, client.ObjectKey{
				Namespace: req.Msg.GetProject(),
				Name:      req.Msg.GetName(),
			}, &kargov1alpha1.Stage{}); err != nil {
				if kubeerr.IsNotFound(err) {
					return connect.NewError(connect.CodeNotFound, err)
				}
				return connect.NewError(connect.CodeInternal, err)
			}
		}

		opts := metav1.ListOptions{}
		if req.Msg.GetName() != "" {
			opts.FieldSelector = fields.OneTermEqualSelector(metav1.ObjectNameField, req.Msg.GetName()).String()
		}
		w, err := stageCli.Namespace(req.Msg.GetProject()).Watch(ctx, opts)
		if err != nil {
			return errors.Wrap(err, "watch stage")
		}
		defer w.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case e, ok := <-w.ResultChan():
				if !ok {
					return nil
				}
				u, ok := e.Object.(*unstructured.Unstructured)
				if !ok {
					return errors.Errorf("unexpected object type %T", e.Object)
				}
				var stage *kargov1alpha1.Stage
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &stage); err != nil {
					return errors.Wrap(err, "from unstructured")
				}
				if err := stream.Send(&svcv1alpha1.WatchStageResponse{
					Stage: typesv1alpha1.ToStageProto(*stage),
					Type:  string(e.Type),
				}); err != nil {
					return errors.Wrap(err, "send response")
				}
			}
		}
	}
}
