package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) GetFreight(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetFreightRequest],
) (*connect.Response[svcv1alpha1.GetFreightResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	alias := req.Msg.GetAlias()
	if (name == "" && alias == "") || (name != "" && alias != "") {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("exactly one of name or alias should not be empty"),
		)
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	// Get the Freight from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "Freight",
		},
	}

	switch {
	case alias != "":
		ul := unstructured.UnstructuredList{
			Object: map[string]any{
				"apiVersion": kargoapi.GroupVersion.String(),
				"kind":       "FreightList",
			},
		}
		if err := s.client.List(
			ctx,
			&ul,
			client.InNamespace(project),
			client.MatchingLabels{
				kargoapi.AliasLabelKey: alias,
			},
		); err != nil {
			return nil, err
		}
		if len(ul.Items) == 0 {
			return nil, connect.NewError(
				connect.CodeNotFound,
				fmt.Errorf("Freight with alias %q not found in namespace %q", alias, project),
			)
		}
		u = ul.Items[0]
	default:
		if err := s.client.Get(ctx, types.NamespacedName{
			Namespace: project,
			Name:      name,
		}, &u); err != nil {
			if client.IgnoreNotFound(err) == nil {
				err = fmt.Errorf("Freight %q not found in namespace %q", name, project)
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			return nil, err
		}
	}

	switch req.Msg.GetFormat() {
	case svcv1alpha1.RawFormat_RAW_FORMAT_JSON, svcv1alpha1.RawFormat_RAW_FORMAT_YAML:
		_, raw, err := objectOrRaw(&u, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetFreightResponse{
			Result: &svcv1alpha1.GetFreightResponse_Raw{
				Raw: raw,
			},
		}), nil
	default:
		f := kargoapi.Freight{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &f); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		obj, _, err := objectOrRaw(&f, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetFreightResponse{
			Result: &svcv1alpha1.GetFreightResponse_Freight{
				Freight: obj,
			},
		}), nil
	}
}
