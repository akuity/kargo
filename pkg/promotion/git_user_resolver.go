package promotion

import (
	"context"
	"fmt"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/controller/git"
)

// signingKeyDataKey is the well-known data key within a Secret that holds the
// GPG private key material.
const signingKeyDataKey = "signingKey"

// GitUserResolver resolves the system-level git user configuration, including
// any signing key. It checks ClusterConfig first, falling back to the
// install-time (env-based) configuration.
type GitUserResolver interface {
	Resolve(ctx context.Context) (git.User, error)
}

// gitUserResolver is the default implementation of GitUserResolver.
type gitUserResolver struct {
	kargoClient     client.Client
	systemNamespace string
	fallback        git.User
}

// NewGitUserResolver returns a GitUserResolver that checks ClusterConfig for
// a signing key Secret reference, falling back to the provided default git
// user (typically populated from environment variables at startup).
func NewGitUserResolver(
	kargoClient client.Client,
	systemNamespace string,
	fallback git.User,
) GitUserResolver {
	return &gitUserResolver{
		kargoClient:     kargoClient,
		systemNamespace: systemNamespace,
		fallback:        fallback,
	}
}

func (r *gitUserResolver) Resolve(
	ctx context.Context,
) (git.User, error) {
	clusterCfg, err := api.GetClusterConfig(ctx, r.kargoClient)
	if err != nil {
		return git.User{}, fmt.Errorf("error getting ClusterConfig: %w", err)
	}
	if clusterCfg == nil || clusterCfg.Spec.GitClient == nil {
		return r.fallback, nil
	}
	gitClient := clusterCfg.Spec.GitClient
	user := git.User{
		Name:  gitClient.Name,
		Email: gitClient.Email,
	}
	if gitClient.SigningKeySecret == nil {
		// ClusterConfig specifies name/email but no signing key. Use those
		// values but preserve any signing key from the fallback.
		user.SigningKeyType = r.fallback.SigningKeyType
		user.SigningKey = r.fallback.SigningKey
		user.SigningKeyPath = r.fallback.SigningKeyPath
		return user, nil
	}
	secret := corev1.Secret{}
	if err = r.kargoClient.Get(
		ctx,
		types.NamespacedName{
			Namespace: r.systemNamespace,
			Name:      gitClient.SigningKeySecret.Name,
		},
		&secret,
	); err != nil {
		return git.User{}, fmt.Errorf(
			"error getting signing key Secret %q: %w",
			gitClient.SigningKeySecret.Name,
			err,
		)
	}
	keyData, ok := secret.Data[signingKeyDataKey]
	if !ok {
		return git.User{}, fmt.Errorf(
			"secret %q does not contain expected key %q",
			gitClient.SigningKeySecret.Name,
			signingKeyDataKey,
		)
	}
	user.SigningKeyType = git.SigningKeyTypeGPG
	user.SigningKey = string(keyData)
	return user, nil
}

// GitUserFromEnv populates a git.User from environment variables. This is
// used as the fallback configuration when no ClusterConfig-based git user is
// available.
func GitUserFromEnv() git.User {
	cfg := struct {
		Name           string `envconfig:"GITCLIENT_NAME"`
		Email          string `envconfig:"GITCLIENT_EMAIL"`
		SigningKeyType string `envconfig:"GITCLIENT_SIGNING_KEY_TYPE"`
		SigningKeyPath string `envconfig:"GITCLIENT_SIGNING_KEY_PATH"`
	}{}
	envconfig.MustProcess("", &cfg)
	return git.User{
		Name:           cfg.Name,
		Email:          cfg.Email,
		SigningKeyType: git.SigningKeyType(cfg.SigningKeyType),
		SigningKeyPath: cfg.SigningKeyPath,
	}
}
