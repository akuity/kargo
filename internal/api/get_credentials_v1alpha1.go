package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libCreds "github.com/akuity/kargo/internal/credentials"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

const redacted = "*** REDACTED ***"

func (s *server) GetCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetCredentialsRequest],
) (*connect.Response[svcv1alpha1.GetCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			fmt.Errorf("secret management is not enabled"),
		)
	}

	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: project,
			Name:      name,
		},
		&secret,
	); err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}

	// If this isn't labeled as repository credentials, return not found.
	var isCredentials bool
	if secret.Labels != nil {
		_, isCredentials = secret.Labels[kargoapi.CredentialTypeLabelKey]
	}
	if !isCredentials {
		return nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"secret %q exists, but is not labeled with %q",
				secret.Name,
				kargoapi.CredentialTypeLabelKey,
			),
		)
	}

	obj, raw, err := objectOrRaw(sanitizeCredentialSecret(secret), req.Msg.GetFormat())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetCredentialsResponse{
			Result: &svcv1alpha1.GetCredentialsResponse_Raw{
				Raw: raw,
			},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetCredentialsResponse{
		Result: &svcv1alpha1.GetCredentialsResponse_Credentials{
			Credentials: obj,
		},
	}), nil
}

// sanitizeCredentialSecret returns a copy of the secret with all values in the
// stringData map redacted except for those with specific keys that are known to
// represent non-sensitive information when used correctly. The primary
// intention, at present, is only to redact the value associated with the
// "password" key, but this approach prevents accidental exposure of the
// password in the event that it has accidentally been assigned to a
// wrong/unknown key, such as "pass" or "passwd". All annotations are also
// redacted because AT LEAST "last-applied-configuration" is a known vector for
// leaking sensitive information and unknown configuration management tools may
// use other annotations in a manner similar to "last-applied-configuration".
// There is no concern over labels because the constraints on label values rule
// out use in a manner similar to that of the "last-applied-configuration"
// annotation.
func sanitizeCredentialSecret(secret corev1.Secret) *corev1.Secret {
	s := secret.DeepCopy()
	s.StringData = make(map[string]string, len(s.Data))
	for k, v := range s.Annotations {
		switch k {
		case kargoapi.AnnotationKeyDescription:
			s.Annotations[k] = v
		default:
			s.Annotations[k] = redacted
		}
	}
	for k, v := range s.Data {
		switch k {
		case libCreds.FieldRepoURL, libCreds.FieldRepoURLIsRegex, libCreds.FieldUsername:
			s.StringData[k] = string(v)
		default:
			s.StringData[k] = redacted
		}
	}
	s.Data = nil
	return s
}
