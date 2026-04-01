package replication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// SetupSecretReconcilerWithManager initializes the Secret replication
// reconciler and registers it with the provided Manager.
func SetupSecretReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	return setupReconcilerWithManager(
		ctx, kargoMgr, cfg, secretAdapter{},
		"shared-secrets-replication-controller",
	)
}

// ---- Secret adapter ----

type secretAdapter struct{}

var _ resourceAdapter = secretAdapter{}

func (secretAdapter) newObject() client.Object   { return &corev1.Secret{} }
func (secretAdapter) newList() client.ObjectList { return &corev1.SecretList{} }

func (secretAdapter) getItems(l client.ObjectList) []client.Object {
	list, ok := l.(*corev1.SecretList)
	if !ok {
		return nil
	}
	items := make([]client.Object, len(list.Items))
	for i := range list.Items {
		items[i] = &list.Items[i]
	}
	return items
}

func (secretAdapter) computeHash(obj client.Object) string {
	s, ok := obj.(*corev1.Secret)
	if !ok {
		return ""
	}
	return computeSecretHash(s)
}

func (secretAdapter) copyFields(dst, src client.Object) {
	d, ok := dst.(*corev1.Secret)
	if !ok {
		return
	}
	s, ok := src.(*corev1.Secret)
	if !ok {
		return
	}
	d.Data = s.Data
	d.Type = s.Type
}

func (secretAdapter) shouldReconcile(obj client.Object) bool {
	s, ok := obj.(*corev1.Secret)
	if !ok {
		return false
	}
	// only reconcile Secrets that are labeled with a credential type,
	// to avoid replicating non-credential Secrets
	_, ok = s.Labels[kargoapi.LabelKeyCredentialType]
	return ok
}

// computeSecretHash returns a deterministic 16-character truncated hex SHA-256
// hash of a Secret's labels, annotations, and data.
func computeSecretHash(secret *corev1.Secret) string {
	h := sha256.New()
	hashMetadata(h, secret.Labels, secret.Annotations)
	h.Write([]byte("data"))
	dataKeys := make([]string, 0, len(secret.Data))
	for k := range secret.Data {
		dataKeys = append(dataKeys, k)
	}
	sort.Strings(dataKeys)
	for _, k := range dataKeys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write(secret.Data[k])
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}
