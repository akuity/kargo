package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetCredentialsRequest],
) (*connect.Response[svcv1alpha1.GetCredentialsResponse], error) {
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
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, errors.Wrap(err, "get secret")
	}

	// If this isn't labeled as repository credentials, return not found.
	var isCredentials bool
	if secret.Labels != nil {
		_, isCredentials = secret.Labels[kargoapi.CredentialTypeLabelKey]
	}
	if !isCredentials {
		return nil, connect.NewError(
			connect.CodeNotFound,
			errors.Errorf("secret %q exists, but is not labeled as credentials", name),
		)
	}

	secret = redactCredentialSecretValues(secret)

	return connect.NewResponse(&svcv1alpha1.GetCredentialsResponse{
		Credentials: typesv1alpha1.ToSecretProto(&secret),
	}), nil
}

// redactCredentialSecretValues returns a copy of the secret with all values in
// the stringData map redacted except for those with specific keys that are
// known to represent non-sensitive information when used correctly. Note: The
// primary intention, at present, is only to redact the value associated with
// the "password" key, but this approach prevents accidental exposure of the
// password in the event that it has accidentally been assigned to a
// wrong/unknown key, such as "pass" or "passwd".
func redactCredentialSecretValues(secret corev1.Secret) corev1.Secret {
	secret.StringData = make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		switch k {
		case "repoURL", "repoURLPattern", "username":
			secret.StringData[k] = string(v)
		default:
			secret.StringData[k] = "*** REDACTED ***"
		}
	}
	secret.Data = nil
	return secret
}
