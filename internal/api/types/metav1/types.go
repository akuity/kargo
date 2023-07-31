package metav1

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	kubemetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	"github.com/akuity/kargo/pkg/api/metav1"
)

func FromListMetaProto(m *metav1.ListMeta) *kubemetav1.ListMeta {
	if m == nil {
		return nil
	}
	return &kubemetav1.ListMeta{
		SelfLink:           m.GetSelfLink(),
		ResourceVersion:    m.GetResourceVersion(),
		Continue:           m.GetContinue(),
		RemainingItemCount: pointer.Int64(m.GetRemainingItemCount()),
	}
}

func FromObjectMetaProto(m *metav1.ObjectMeta) *kubemetav1.ObjectMeta {
	if m == nil {
		return nil
	}
	var deletionTimestamp *kubemetav1.Time
	if m.GetDeletionTimestamp() != nil {
		dts := kubemetav1.NewTime(m.GetDeletionTimestamp().AsTime())
		deletionTimestamp = &dts
	}
	ownerRefs := make([]kubemetav1.OwnerReference, len(m.GetOwnerReferences()))
	for idx, r := range m.GetOwnerReferences() {
		ownerRefs[idx] = *FromOwnerReferenceProto(r)
	}
	managedFields := make([]kubemetav1.ManagedFieldsEntry, len(m.GetManagedFields()))
	for idx, f := range m.GetManagedFields() {
		managedFields[idx] = *FromManagedFieldsEntryProto(f)
	}
	return &kubemetav1.ObjectMeta{
		Name:                       m.GetName(),
		GenerateName:               m.GetGenerateName(),
		Namespace:                  m.GetNamespace(),
		SelfLink:                   m.GetSelfLink(),
		UID:                        types.UID(m.GetUid()),
		ResourceVersion:            m.GetResourceVersion(),
		Generation:                 m.GetGeneration(),
		CreationTimestamp:          kubemetav1.NewTime(m.GetCreationTimestamp().AsTime()),
		DeletionTimestamp:          deletionTimestamp,
		DeletionGracePeriodSeconds: pointer.Int64(m.GetDeletionGracePeriodSeconds()),
		Labels:                     m.GetLabels(),
		Annotations:                m.GetAnnotations(),
		OwnerReferences:            ownerRefs,
		Finalizers:                 m.GetFinalizers(),
		ManagedFields:              managedFields,
	}
}

func FromOwnerReferenceProto(r *metav1.OwnerReference) *kubemetav1.OwnerReference {
	if r == nil {
		return nil
	}
	return &kubemetav1.OwnerReference{
		APIVersion:         r.GetApiVersion(),
		Kind:               r.GetKind(),
		Name:               r.GetName(),
		UID:                types.UID(r.GetUid()),
		Controller:         pointer.Bool(r.GetController()),
		BlockOwnerDeletion: pointer.Bool(r.GetBlockOwnerDeletion()),
	}
}

func FromFieldsV1Proto(f *metav1.FieldsV1) *kubemetav1.FieldsV1 {
	if f == nil {
		return nil
	}
	return &kubemetav1.FieldsV1{
		Raw: f.GetRaw(),
	}
}

func FromManagedFieldsEntryProto(e *metav1.ManagedFieldsEntry) *kubemetav1.ManagedFieldsEntry {
	if e == nil {
		return nil
	}
	t := kubemetav1.NewTime(e.Time.AsTime())
	return &kubemetav1.ManagedFieldsEntry{
		Manager:     e.GetManager(),
		Operation:   kubemetav1.ManagedFieldsOperationType(e.GetOperation()),
		APIVersion:  e.GetApiVersion(),
		Time:        &t,
		FieldsType:  e.GetFieldsType(),
		FieldsV1:    FromFieldsV1Proto(e.GetFieldsV1()),
		Subresource: e.GetSubresource(),
	}
}

func ToListMetaProto(m kubemetav1.ListMeta) *metav1.ListMeta {
	return &metav1.ListMeta{
		SelfLink:           proto.String(m.GetSelfLink()),
		ResourceVersion:    proto.String(m.GetResourceVersion()),
		Continue:           proto.String(m.GetContinue()),
		RemainingItemCount: m.GetRemainingItemCount(),
	}
}

func ToObjectMetaProto(m kubemetav1.ObjectMeta) *metav1.ObjectMeta {
	var deletionTimestamp *timestamppb.Timestamp
	if m.GetDeletionTimestamp() != nil {
		deletionTimestamp = timestamppb.New(m.GetDeletionTimestamp().Time)
	}
	ownerRefs := make([]*metav1.OwnerReference, len(m.GetOwnerReferences()))
	for idx, r := range m.GetOwnerReferences() {
		ownerRefs[idx] = ToOwnerReferenceProto(r)
	}
	managedFields := make([]*metav1.ManagedFieldsEntry, len(m.GetManagedFields()))
	for idx, f := range m.GetManagedFields() {
		managedFields[idx] = ToManagedFieldsEntryProto(f)
	}
	return &metav1.ObjectMeta{
		Name:                       pointer.String(m.GetName()),
		GenerateName:               pointer.String(m.GetGenerateName()),
		Namespace:                  pointer.String(m.GetNamespace()),
		SelfLink:                   pointer.String(m.GetSelfLink()),
		Uid:                        pointer.String(string(m.GetUID())),
		ResourceVersion:            pointer.String(m.GetResourceVersion()),
		Generation:                 pointer.Int64(m.GetGeneration()),
		CreationTimestamp:          timestamppb.New(m.GetCreationTimestamp().Time),
		DeletionTimestamp:          deletionTimestamp,
		DeletionGracePeriodSeconds: m.GetDeletionGracePeriodSeconds(),
		Labels:                     m.GetLabels(),
		Annotations:                m.GetAnnotations(),
		OwnerReferences:            ownerRefs,
		Finalizers:                 m.GetFinalizers(),
		ManagedFields:              managedFields,
	}
}

func ToOwnerReferenceProto(r kubemetav1.OwnerReference) *metav1.OwnerReference {
	return &metav1.OwnerReference{
		ApiVersion:         proto.String(r.APIVersion),
		Kind:               proto.String(r.Kind),
		Name:               proto.String(r.Name),
		Uid:                proto.String(string(r.UID)),
		Controller:         r.Controller,
		BlockOwnerDeletion: r.BlockOwnerDeletion,
	}
}

func ToManagedFieldsEntryProto(e kubemetav1.ManagedFieldsEntry) *metav1.ManagedFieldsEntry {
	var t *timestamppb.Timestamp
	if e.Time != nil {
		t = timestamppb.New(e.Time.Time)
	}
	var fieldsV1 *metav1.FieldsV1
	if e.FieldsV1 != nil {
		fieldsV1 = ToFieldsV1Proto(*e.FieldsV1)
	}
	return &metav1.ManagedFieldsEntry{
		Manager:     proto.String(e.Manager),
		Operation:   proto.String(string(e.Operation)),
		ApiVersion:  proto.String(e.APIVersion),
		Time:        t,
		FieldsType:  proto.String(e.FieldsType),
		FieldsV1:    fieldsV1,
		Subresource: proto.String(e.Subresource),
	}
}

func ToFieldsV1Proto(f kubemetav1.FieldsV1) *metav1.FieldsV1 {
	return &metav1.FieldsV1{
		Raw: f.Raw,
	}
}
