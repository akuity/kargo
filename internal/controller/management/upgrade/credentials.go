package upgrade

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libCreds "github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

const (
	SecretTypeLabelKey  = "kargo.akuity.io/secret-type" // nolint: gosec
	repoLabelValue      = "repository"
	repoCredsLabelValue = "repo-creds" // nolint: gosec
)

// credentialsReconciler reconciles credentials (Secrets) to upgrade them from
// v0.4-compatible to v0.5-compatible.
type credentialsReconciler struct {
	client client.Client
}

// SetupCredentialsReconcilerWithManager initializes a credentialsReconciler
// and registers it with the provided Manager.
func SetupCredentialsReconcilerWithManager(mgr manager.Manager) error {
	labelPred, err := predicate.LabelSelectorPredicate(
		metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{ // All Secrets that use the old SecretTypeLabelKey
					Key:      SecretTypeLabelKey,
					Operator: metav1.LabelSelectorOpExists,
				},
			},
		},
	)
	if err != nil {
		return err
	}
	_, err = ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					return false
				},
			},
		).
		WithEventFilter(labelPred).
		Build(&credentialsReconciler{
			client: mgr.GetClient(),
		})
	return err
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (f *credentialsReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"secret":    req.NamespacedName.Name,
	})
	logger.Debug("reconciling credentials (Secret)")

	// Find the Secret
	secret := &corev1.Secret{}
	if err := f.client.Get(ctx, req.NamespacedName, secret); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Update the credentials to be v0.5-compatible
	transformCredentialsSecret(secret)

	if err := f.client.Update(ctx, secret); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		Requeue: false,
	}, nil
}

func transformCredentialsSecret(secret *corev1.Secret) {
	// This label is guaranteed to exist because the reconciler uses a predicate
	// that requires it.
	credType := secret.Labels[SecretTypeLabelKey]
	url := string(secret.Data["url"])
	if credType == repoCredsLabelValue {
		url = fmt.Sprintf(`^%s(/.*)?$`, strings.TrimSuffix(url, "/"))
	}

	secret.StringData = map[string]string{
		libCreds.FieldRepoURL:  url,
		libCreds.FieldUsername: string(secret.Data[libCreds.FieldUsername]),
		libCreds.FieldPassword: string(secret.Data[libCreds.FieldPassword]),
	}
	if credType == repoCredsLabelValue {
		secret.StringData[libCreds.FieldRepoURLIsRegex] = "true"
	}

	delete(secret.Labels, SecretTypeLabelKey)
	secret.Labels[kargoapi.CredentialTypeLabelKey] = string(secret.Data["type"])

	secret.Data = nil
}
