package svcv1alpha1

import (
	"maps"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// toK8sObjectMeta converts a simplified protobuf ObjectMeta to a Kubernetes ObjectMeta.
func toK8sObjectMeta(om *ObjectMeta) metav1.ObjectMeta {
	if om == nil {
		return metav1.ObjectMeta{}
	}

	k8sMeta := metav1.ObjectMeta{}

	if om.Name != nil {
		k8sMeta.Name = *om.Name
	}
	if om.Namespace != nil {
		k8sMeta.Namespace = *om.Namespace
	}
	if om.Uid != nil {
		k8sMeta.UID = types.UID(*om.Uid)
	}
	if om.ResourceVersion != nil {
		k8sMeta.ResourceVersion = *om.ResourceVersion
	}
	if om.Generation != nil {
		k8sMeta.Generation = *om.Generation
	}
	if om.Labels != nil {
		k8sMeta.Labels = make(map[string]string, len(om.Labels))
		maps.Copy(k8sMeta.Labels, om.Labels)
	}
	if om.Annotations != nil {
		k8sMeta.Annotations = make(map[string]string, len(om.Annotations))
		maps.Copy(k8sMeta.Annotations, om.Annotations)
	}

	return k8sMeta
}

// fromK8sObjectMeta converts a Kubernetes ObjectMeta to a simplified protobuf ObjectMeta.
func fromK8sObjectMeta(k8sMeta *metav1.ObjectMeta) *ObjectMeta {
	if k8sMeta == nil {
		return nil
	}

	om := &ObjectMeta{}

	if k8sMeta.Name != "" {
		om.Name = &k8sMeta.Name
	}
	if k8sMeta.Namespace != "" {
		om.Namespace = &k8sMeta.Namespace
	}
	if k8sMeta.UID != "" {
		uid := string(k8sMeta.UID)
		om.Uid = &uid
	}
	if k8sMeta.ResourceVersion != "" {
		om.ResourceVersion = &k8sMeta.ResourceVersion
	}
	if k8sMeta.Generation != 0 {
		om.Generation = &k8sMeta.Generation
	}
	if k8sMeta.Labels != nil {
		om.Labels = make(map[string]string, len(k8sMeta.Labels))
		maps.Copy(om.Labels, k8sMeta.Labels)
	}
	if k8sMeta.Annotations != nil {
		om.Annotations = make(map[string]string, len(k8sMeta.Annotations))
		maps.Copy(om.Annotations, k8sMeta.Annotations)
	}

	return om
}

// ToK8sConfigMap converts a protobuf ConfigMap to a Kubernetes ConfigMap.
func (c *ConfigMap) ToK8sConfigMap() *corev1.ConfigMap {
	if c == nil {
		return nil
	}

	cm := &corev1.ConfigMap{}

	cm.ObjectMeta = toK8sObjectMeta(c.Metadata)

	if c.Immutable != nil {
		cm.Immutable = c.Immutable
	}

	if c.Data != nil {
		cm.Data = make(map[string]string, len(c.Data))
		maps.Copy(cm.Data, c.Data)
	}

	if c.BinaryData != nil {
		cm.BinaryData = make(map[string][]byte, len(c.BinaryData))
		maps.Copy(cm.BinaryData, c.BinaryData)
	}

	return cm
}

// FromK8sConfigMap converts a Kubernetes ConfigMap to a protobuf ConfigMap.
func FromK8sConfigMap(cm *corev1.ConfigMap) *ConfigMap {
	if cm == nil {
		return nil
	}

	c := &ConfigMap{}

	// Copy ObjectMeta
	c.Metadata = fromK8sObjectMeta(&cm.ObjectMeta)

	// Copy Immutable
	if cm.Immutable != nil {
		c.Immutable = cm.Immutable
	}

	// Copy Data
	if cm.Data != nil {
		c.Data = make(map[string]string, len(cm.Data))
		maps.Copy(c.Data, cm.Data)
	}

	// Copy BinaryData
	if cm.BinaryData != nil {
		c.BinaryData = make(map[string][]byte, len(cm.BinaryData))
		maps.Copy(c.BinaryData, cm.BinaryData)
	}

	return c
}

// ToK8sSecret converts a protobuf Secret to a Kubernetes Secret.
func (s *Secret) ToK8sSecret() *corev1.Secret {
	if s == nil {
		return nil
	}

	secret := &corev1.Secret{}

	secret.ObjectMeta = toK8sObjectMeta(s.Metadata)

	if s.Immutable != nil {
		secret.Immutable = s.Immutable
	}

	if s.Data != nil {
		secret.Data = make(map[string][]byte, len(s.Data))
		maps.Copy(secret.Data, s.Data)
	}

	if s.StringData != nil {
		secret.StringData = make(map[string]string, len(s.StringData))
		maps.Copy(secret.StringData, s.StringData)
	}

	if s.Type != nil {
		t := corev1.SecretType(*s.Type)
		secret.Type = t
	}

	return secret
}

// FromK8sSecret converts a Kubernetes Secret to a protobuf Secret.
func FromK8sSecret(secret *corev1.Secret) *Secret {
	if secret == nil {
		return nil
	}

	s := &Secret{}

	// Copy ObjectMeta
	s.Metadata = fromK8sObjectMeta(&secret.ObjectMeta)

	// Copy Immutable
	if secret.Immutable != nil {
		s.Immutable = secret.Immutable
	}

	// Copy Data
	if secret.Data != nil {
		s.Data = make(map[string][]byte, len(secret.Data))
		maps.Copy(s.Data, secret.Data)
	}

	// Copy StringData
	if secret.StringData != nil {
		s.StringData = make(map[string]string, len(secret.StringData))
		maps.Copy(s.StringData, secret.StringData)
	}

	// Copy Type
	if secret.Type != "" {
		t := string(secret.Type)
		s.Type = &t
	}

	return s
}
