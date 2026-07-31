package utils

import (
	"context"
	"errors"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"

	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
)

// GitCreds holds the git credentials substituted into git-driven suite
// fixtures.
type GitCreds struct {
	RepoURL  string
	Username string
	Password string
}

// RequireGitCreds returns credentials for the demo GitOps repository fork,
// sourced from the test environment (kargo_demo_gitops_repo + git_pat). The
// username is derived from the repository owner; GitHub authenticates via the
// PAT regardless of the username. It fails the test if either value is missing.
func RequireGitCreds(ctx context.Context, t *testing.T) GitCreds {
	repoURL := requireKargoDemoRepo(ctx, t)
	patVal, err := envfuncs.GetEnv(ctx, []string{"context", "git_pat"})
	if err != nil {
		t.Fatalf("cannot get git_pat %v", err)
	}
	pat, ok := patVal.(string)
	if !ok {
		t.Fatalf("git_pat is not a string: %v", patVal)
	}
	return GitCreds{
		RepoURL:  repoURL,
		Username: gitRepoOwner(repoURL),
		Password: pat,
	}
}

// gitRepoOwner extracts the owner segment from a Git HTTPS URL, e.g.
// https://github.com/octocat/repo.git -> "octocat".
func gitRepoOwner(repoURL string) string {
	owner, _, _ := gitHubOwnerRepo(repoURL)
	return owner
}

// UpdateGitCredentialsSecret returns a DecodeOption that rewrites the repoURL,
// username and password of the named git credentials Secret. Core types decode
// into their typed representation, so this operates on a *corev1.Secret.
func UpdateGitCredentialsSecret(name, repoURL, username, password string) decoder.DecodeOption {
	return MutateOptionFor("Secret", name, func(obj k8s.Object) error {
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			return fmt.Errorf("object %q is not a *corev1.Secret", name)
		}
		if secret.StringData == nil {
			secret.StringData = map[string]string{}
		}
		secret.StringData["repoURL"] = repoURL
		secret.StringData["username"] = username
		secret.StringData["password"] = password
		return nil
	})
}

// UpdateStagePromotionVar returns a DecodeOption that rewrites the value of the
// promotion variable named key in the named Stage's promotionTemplate. An empty
// stageName matches every Stage.
func UpdateStagePromotionVar(stageName, key, val string) decoder.DecodeOption {
	return MutateAsUnstructuredOptionFor("Stage", stageName, func(unstr runtime.Unstructured) error {
		data := unstr.UnstructuredContent()
		spec, ok := data["spec"].(map[string]any)
		if !ok {
			return errors.New("Stage spec is not a map")
		}
		promoTmpl, ok := spec["promotionTemplate"].(map[string]any)
		if !ok {
			return errors.New("Stage spec.promotionTemplate is not a map")
		}
		promoSpec, ok := promoTmpl["spec"].(map[string]any)
		if !ok {
			return errors.New("Stage spec.promotionTemplate.spec is not a map")
		}
		vars, ok := promoSpec["vars"].([]any)
		if !ok {
			// No vars to update.
			return nil
		}
		for _, v := range vars {
			if vm, ok := v.(map[string]any); ok && vm["name"] == key {
				vm["value"] = val
			}
		}

		unstr.SetUnstructuredContent(data)
		return nil
	})
}
