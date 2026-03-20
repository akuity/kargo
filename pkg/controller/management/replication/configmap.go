package replication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupConfigMapReconcilerWithManager initializes the ConfigMap replication
// reconciler and registers it with the provided Manager.
func SetupConfigMapReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	return setupReconcilerWithManager(
		ctx, kargoMgr, cfg, configMapAdapter{},
		"shared-configmaps-replication-controller",
	)
}

// ---- ConfigMap adapter ----

type configMapAdapter struct{}

var _ resourceAdapter = configMapAdapter{}

func (configMapAdapter) newObject() client.Object   { return &corev1.ConfigMap{} }
func (configMapAdapter) newList() client.ObjectList { return &corev1.ConfigMapList{} }

func (configMapAdapter) getItems(l client.ObjectList) []client.Object {
	list, ok := l.(*corev1.ConfigMapList)
	if !ok {
		return nil
	}
	items := make([]client.Object, len(list.Items))
	for i := range list.Items {
		items[i] = &list.Items[i]
	}
	return items
}

func (configMapAdapter) computeHash(obj client.Object) string {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return ""
	}
	return computeConfigMapHash(cm)
}

func (configMapAdapter) copyFields(dst, src client.Object) {
	d, ok := dst.(*corev1.ConfigMap)
	if !ok {
		return
	}
	s, ok := src.(*corev1.ConfigMap)
	if !ok {
		return
	}
	d.Data = s.Data
	d.BinaryData = s.BinaryData
}

func (configMapAdapter) shouldReconcile(_ client.Object) bool {
	// All ConfigMaps in the shared resources namespace should be reconciled
	return true
}

// computeConfigMapHash returns a deterministic 16-character truncated hex
// SHA-256 hash of a ConfigMap's labels, annotations, data, and binaryData.
func computeConfigMapHash(cm *corev1.ConfigMap) string {
	h := sha256.New()
	hashMetadata(h, cm.Labels, cm.Annotations)
	h.Write([]byte("data"))
	dataKeys := make([]string, 0, len(cm.Data))
	for k := range cm.Data {
		dataKeys = append(dataKeys, k)
	}
	sort.Strings(dataKeys)
	for _, k := range dataKeys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write([]byte(cm.Data[k]))
		h.Write([]byte{0})
	}
	h.Write([]byte("binaryData"))
	binaryDataKeys := make([]string, 0, len(cm.BinaryData))
	for k := range cm.BinaryData {
		binaryDataKeys = append(binaryDataKeys, k)
	}
	sort.Strings(binaryDataKeys)
	for _, k := range binaryDataKeys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write(cm.BinaryData[k])
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}
