package v1alpha1

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Shard,type=string,JSONPath=`.spec.shard`
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Warehouse is a source of Freight.
type Warehouse struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes sources of artifacts.
	//
	// +kubebuilder:validation:Required
	Spec WarehouseSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the Warehouse's most recently observed state.
	Status WarehouseStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// GetInterval calculates and returns interval time remaining until the next
// requeue should occur. If the interval has passed, it returns a short duration
// to ensure the Warehouse is requeued promptly.
func (w *Warehouse) GetInterval(minInterval time.Duration) time.Duration {
	effectiveInterval := w.Spec.Interval.Duration
	if effectiveInterval < minInterval {
		effectiveInterval = minInterval
	}

	if w.Status.DiscoveredArtifacts == nil || w.Status.DiscoveredArtifacts.DiscoveredAt.IsZero() {
		return effectiveInterval
	}

	if interval := w.Status.DiscoveredArtifacts.DiscoveredAt.
		Add(effectiveInterval).
		Sub(metav1.Now().Time); interval > 0 {
		return interval
	}
	return 100 * time.Millisecond
}

func (w *Warehouse) GetStatus() *WarehouseStatus {
	return &w.Status
}

// WarehouseSpec describes sources of versioned artifacts to be included in
// Freight produced by this Warehouse.
type WarehouseSpec struct {
	// Shard is the name of the shard that this Warehouse belongs to. This is an
	// optional field. If not specified, the Warehouse will belong to the default
	// shard. A defaulting webhook will sync this field with the value of the
	// kargo.akuity.io/shard label. When the shard label is not present or differs
	// from the value of this field, the defaulting webhook will set the label to
	// the value of this field. If the shard label is present and this field is
	// empty, the defaulting webhook will set the value of this field to the value
	// of the shard label.
	Shard string `json:"shard,omitempty" protobuf:"bytes,2,opt,name=shard"`
	// Interval is the reconciliation interval for this Warehouse. On each
	// reconciliation, the Warehouse will discover new artifacts and optionally
	// produce new Freight. This field is optional. When left unspecified, the
	// field is implicitly treated as if its value were "5m0s".
	//
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(s|m|h))+$`
	// +kubebuilder:default="5m0s"
	// +akuity:test-kubebuilder-pattern=Duration
	Interval metav1.Duration `json:"interval" protobuf:"bytes,4,opt,name=interval"`
	// FreightCreationPolicy describes how Freight is created by this Warehouse.
	// This field is optional. When left unspecified, the field is implicitly
	// treated as if its value were "Automatic".
	//
	// Accepted values:
	//
	// - "Automatic": New Freight is created automatically when any new artifact
	//   is discovered.
	// - "Manual": New Freight is never created automatically.
	//
	// +kubebuilder:default=Automatic
	// +kubebuilder:validation:Optional
	FreightCreationPolicy FreightCreationPolicy `json:"freightCreationPolicy" protobuf:"bytes,3,opt,name=freightCreationPolicy"`
	// Subscriptions describes sources of artifacts to be included in Freight
	// produced by this Warehouse.
	//
	// +kubebuilder:validation:MinItems=1
	Subscriptions []apiextensionsv1.JSON `json:"subscriptions" protobuf:"bytes,1,rep,name=subscriptions"`
	// InternalSubscriptions is an internal, typed representation of the
	// Subscriptions field. When a WarehouseSpec is unmarshaled, this field is
	// populated from the JSON in the Subscriptions field. When a WarehouseSpec is
	// marshaled, the contents of this field are marshaled into JSON and used to
	// populate the Subscriptions field.
	//
	// Note(krancour): The existence of this field is a short-term workaround that
	// has allowed the Subscriptions field to become raw JSON without forcing us
	// to immediately refactor all existing code that depends on typed
	// RepoSubscription objects. THIS FIELD MAY BE REMOVED WITHOUT NOTICE IN A
	// FUTURE RELEASE.
	//
	// +kubebuilder:validation:Optional
	InternalSubscriptions []RepoSubscription `json:"-" protobuf:"-"`
	// FreightCreationCriteria defines criteria that must be satisfied for Freight
	// to be created automatically from new artifacts following discovery. This
	// field has no effect when the FreightCreationPolicy is `Manual`.
	//
	// +kubebuilder:validation:Optional
	FreightCreationCriteria *FreightCreationCriteria `json:"freightCreationCriteria,omitempty" protobuf:"bytes,5,opt,name=freightCreationCriteria"`
}

var legacySubscriptionTypes = []string{"chart", "git", "image"}

// UnmarshalJSON unmarshals the JSON data into WarehouseSpec, converting the
// JSON from the Subscriptions field into typed RepoSubscription objects in
// InternalSubscriptions. Any JSON object with a top-level key other than "git",
// "image", or "chart" is unpacked into a (generic) Subscription with the key as
// the ArtifactType.
func (w *WarehouseSpec) UnmarshalJSON(data []byte) error {
	type warehouseSpecAlias WarehouseSpec
	aux := &struct {
		Subscriptions []apiextensionsv1.JSON `json:"subscriptions"`
		*warehouseSpecAlias
	}{
		warehouseSpecAlias: (*warehouseSpecAlias)(w),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Store the JSON subscriptions
	w.Subscriptions = aux.Subscriptions

	// Convert JSON Subscriptions to typed RepoSubscription objects
	if len(aux.Subscriptions) > 0 {
		w.InternalSubscriptions = make([]RepoSubscription, len(aux.Subscriptions))
		for i := range aux.Subscriptions {
			// First, unmarshal as a map to check for keys
			rawMap := make(map[string]json.RawMessage)
			if err := json.Unmarshal(aux.Subscriptions[i].Raw, &rawMap); err != nil {
				return err
			}

			// Validate that the subscription is an object with exactly one top-level
			// key
			if len(rawMap) != 1 {
				return fmt.Errorf(
					"subscription at index %d must be an object with exactly one "+
						"top-level field, but has %d fields",
					i, len(rawMap),
				)
			}

			// Get the single key
			var key string
			for k := range rawMap {
				key = k
				break // This unnecessary, but it makes it clear what we're doing
			}

			// Check for known keys (git, image, chart)
			if slices.Contains(legacySubscriptionTypes, key) {
				// Known subscription type - unmarshal normally
				if err := json.Unmarshal(
					aux.Subscriptions[i].Raw,
					&w.InternalSubscriptions[i],
				); err != nil {
					return err
				}
			} else {
				// Generic subscription - unpack with key as ArtifactType
				var sub Subscription
				if err := json.Unmarshal(rawMap[key], &sub); err != nil {
					return err
				}
				sub.SubscriptionType = key
				w.InternalSubscriptions[i].Subscription = &sub
			}
		}
		w.Subscriptions = nil // Clear to avoid confusion
	}

	return nil
}

// MarshalJSON marshals the WarehouseSpec to JSON, converting the
// InternalSubscriptions slice into JSON in the Subscriptions field. For
// Subscription objects, the Kind field is used as the top-level key.
func (w WarehouseSpec) MarshalJSON() ([]byte, error) {
	type warehouseSpecAlias WarehouseSpec
	aux := &struct {
		Subscriptions []apiextensionsv1.JSON `json:"subscriptions"`
		*warehouseSpecAlias
	}{
		warehouseSpecAlias: (*warehouseSpecAlias)(&w),
	}

	// Convert InternalSubscriptions to JSON
	if len(w.InternalSubscriptions) > 0 {
		aux.Subscriptions = make([]apiextensionsv1.JSON, len(w.InternalSubscriptions))
		for i := range w.InternalSubscriptions {
			sub := w.InternalSubscriptions[i]

			// Count how many subscription types are set
			typesSet := 0
			if sub.Git != nil {
				typesSet++
			}
			if sub.Image != nil {
				typesSet++
			}
			if sub.Chart != nil {
				typesSet++
			}
			if sub.Subscription != nil {
				typesSet++
			}

			// Validate that exactly one type is set
			if typesSet != 1 {
				return nil, fmt.Errorf(
					"subscription at index %d must have exactly one of Git, Image, "+
						"Chart, or Subscription set, but has %d",
					i, typesSet,
				)
			}

			// If this is a generic Subscription, wrap it with its Kind as the key
			if sub.Subscription != nil {
				kind := sub.Subscription.SubscriptionType
				if kind == "" {
					return nil, fmt.Errorf(
						"subscription at index %d has empty SubscriptionType field", i,
					)
				}
				genericJSON, err := json.Marshal(sub.Subscription)
				if err != nil {
					return nil, err
				}
				// Wrap in an object with the Kind as the key
				wrapper := map[string]json.RawMessage{
					kind: json.RawMessage(genericJSON),
				}
				jsonData, err := json.Marshal(wrapper)
				if err != nil {
					return nil, err
				}
				aux.Subscriptions[i] = apiextensionsv1.JSON{Raw: jsonData}
			} else {
				// For Git, Image, Chart subscriptions, marshal directly
				jsonData, err := json.Marshal(sub)
				if err != nil {
					return nil, err
				}
				aux.Subscriptions[i] = apiextensionsv1.JSON{Raw: jsonData}
			}
		}
	}

	// Clear the internal field from the copy to avoid duplication in output
	w.InternalSubscriptions = nil

	return json.Marshal(aux)
}

// FreightCreationPolicy defines how Freight is created by a Warehouse.
// +kubebuilder:validation:Enum={Automatic,Manual}
type FreightCreationPolicy string

const (
	// FreightCreationPolicyAutomatic indicates that Freight is created automatically.
	FreightCreationPolicyAutomatic FreightCreationPolicy = "Automatic"
	// FreightCreationPolicyManual indicates that Freight is created manually.
	FreightCreationPolicyManual FreightCreationPolicy = "Manual"
)

// FreightCreationCriteria defines criteria that must be satisfied for Freight
// to be created automatically from new artifacts following discovery.
type FreightCreationCriteria struct {
	// Expression is an expr-lang expression that must evaluate to true for
	// Freight to be created automatically from new artifacts following discovery.
	Expression string `json:"expression,omitempty" protobuf:"bytes,1,opt,name=expression"`
}

// RepoSubscription describes a subscription to ONE OF a Git repository, a
// container image repository, a Helm chart repository, or something else.
type RepoSubscription struct {
	// Git describes a subscriptions to a Git repository.
	Git *GitSubscription `json:"git,omitempty" protobuf:"bytes,1,opt,name=git"`
	// Image describes a subscription to container image repository.
	Image *ImageSubscription `json:"image,omitempty" protobuf:"bytes,2,opt,name=image"`
	// Chart describes a subscription to a Helm chart repository.
	Chart *ChartSubscription `json:"chart,omitempty" protobuf:"bytes,3,opt,name=chart"`
	// Subscription describes a subscription to something that is not a Git, container
	// image, or Helm chart repository.
	Subscription *Subscription `json:"subscription,omitempty" protobuf:"bytes,4,opt,name=subscription"`
}

// Subscription represents a subscription to some kind of artifact repository.
type Subscription struct {
	// SubscriptionType specifies the kind of subscription this is.
	//
	// +kubebuilder:validation:MinLength=1
	SubscriptionType string `json:"subscriptionType" protobuf:"bytes,1,opt,name=subscriptionType"`
	// Name is a unique (with respect to a Warehouse) name used for identifying
	// this subscription.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
	// Config is a JSON object containing opaque configuration for this
	// subscription. (It must be an object. It may not be a list or a scalar
	// value.) This is only understood by a corresponding Subscriber
	// implementation for the ArtifactType.
	//
	// +optional
	Config *apiextensionsv1.JSON `json:"config,omitempty" protobuf:"bytes,3,opt,name=config"`
	// DiscoveryLimit is an optional limit on the number of artifacts that can
	// be discovered for this subscription.
	//
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=20
	DiscoveryLimit int32 `json:"discoveryLimit,omitempty" protobuf:"varint,4,opt,name=discoveryLimit"`
}

// WarehouseStatus describes a Warehouse's most recently observed state.
type WarehouseStatus struct {
	// Conditions contains the last observations of the Warehouse's current
	// state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge" protobuf:"bytes,9,rep,name=conditions"`
	// LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh
	// annotation that was handled by the controller. This field can be used to
	// determine whether the request to refresh the resource has been handled.
	// +optional
	LastHandledRefresh string `json:"lastHandledRefresh,omitempty" protobuf:"bytes,6,opt,name=lastHandledRefresh"`
	// ObservedGeneration represents the .metadata.generation that this Warehouse
	// was reconciled against.
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,4,opt,name=observedGeneration"`
	// LastFreightID is a reference to the system-assigned identifier (name) of
	// the most recent Freight produced by the Warehouse.
	LastFreightID string `json:"lastFreightID,omitempty" protobuf:"bytes,8,opt,name=lastFreightID"`
	// DiscoveredArtifacts holds the artifacts discovered by the Warehouse.
	DiscoveredArtifacts *DiscoveredArtifacts `json:"discoveredArtifacts,omitempty" protobuf:"bytes,7,opt,name=discoveredArtifacts"`
}

// GetConditions implements the conditions.Getter interface.
func (w *WarehouseStatus) GetConditions() []metav1.Condition {
	return w.Conditions
}

// SetConditions implements the conditions.Setter interface.
func (w *WarehouseStatus) SetConditions(conditions []metav1.Condition) {
	w.Conditions = conditions
}

// DiscoveredArtifacts holds the artifacts discovered by the Warehouse for its
// subscriptions.
type DiscoveredArtifacts struct {
	// DiscoveredAt is the time at which the Warehouse discovered the artifacts.
	//
	// +optional
	DiscoveredAt metav1.Time `json:"discoveredAt" protobuf:"bytes,4,opt,name=discoveredAt"`
	// Git holds the commits discovered by the Warehouse for the Git
	// subscriptions.
	//
	// +optional
	Git []GitDiscoveryResult `json:"git,omitempty" protobuf:"bytes,1,rep,name=git"`
	// Images holds the image references discovered by the Warehouse for the
	// image subscriptions.
	//
	// +optional
	Images []ImageDiscoveryResult `json:"images,omitempty" protobuf:"bytes,2,rep,name=images"`
	// Charts holds the charts discovered by the Warehouse for the chart
	// subscriptions.
	//
	// +optional
	Charts []ChartDiscoveryResult `json:"charts,omitempty" protobuf:"bytes,3,rep,name=charts"`
	// Results holds the artifact references discovered by the Warehouse.
	//
	// +optional
	Results []DiscoveryResult `json:"results,omitempty" protobuf:"bytes,5,rep,name=results"`
}

// GitDiscoveryResult represents the result of a Git discovery operation for a
// GitSubscription.
type GitDiscoveryResult struct {
	// RepoURL is the repository URL of the GitSubscription.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`(?:^(ssh|https?)://(?:([\w-]+)(:(.+))?@)?([\w-]+(?:\.[\w-]+)*)(?::(\d{1,5}))?(/.*)$)|(?:^([\w-]+)@([\w+]+(?:\.[\w-]+)*):(/?.*))`
	// +akuity:test-kubebuilder-pattern=GitRepoURLPattern
	RepoURL string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	// Commits is a list of commits discovered by the Warehouse for the
	// GitSubscription. An empty list indicates that the discovery operation was
	// successful, but no commits matching the GitSubscription criteria were found.
	//
	// +optional
	Commits []DiscoveredCommit `json:"commits" protobuf:"bytes,2,rep,name=commits"`
}

// DiscoveredCommit represents a commit discovered by a Warehouse for a
// GitSubscription.
type DiscoveredCommit struct {
	// ID is the identifier of the commit. This typically is a SHA-1 hash.
	//
	// +kubebuilder:validation:MinLength=1
	ID string `json:"id,omitempty" protobuf:"bytes,1,opt,name=id"`
	// Branch is the branch in which the commit was found. This field is
	// optional, and populated based on the CommitSelectionStrategy of the
	// GitSubscription.
	Branch string `json:"branch,omitempty" protobuf:"bytes,2,opt,name=branch"`
	// Tag is the tag that resolved to this commit. This field is optional, and
	// populated based on the CommitSelectionStrategy of the GitSubscription.
	Tag string `json:"tag,omitempty" protobuf:"bytes,3,opt,name=tag"`
	// Subject is the subject of the commit (i.e. the first line of the commit
	// message).
	Subject string `json:"subject,omitempty" protobuf:"bytes,4,opt,name=subject"`
	// Author is the author of the commit.
	Author string `json:"author,omitempty" protobuf:"bytes,5,opt,name=author"`
	// Committer is the person who committed the commit.
	Committer string `json:"committer,omitempty" protobuf:"bytes,6,opt,name=committer"`
	// CreatorDate is the commit creation date as specified by the commit, or
	// the tagger date if the commit belongs to an annotated tag.
	CreatorDate *metav1.Time `json:"creatorDate,omitempty" protobuf:"bytes,7,opt,name=creatorDate"`
}

// ImageDiscoveryResult represents the result of an image discovery operation
// for an ImageSubscription.
type ImageDiscoveryResult struct {
	// RepoURL is the repository URL of the image, as specified in the
	// ImageSubscription.
	//
	// +kubebuilder:validation:MinLength=1
	RepoURL string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	// Platform is the target platform constraint of the ImageSubscription
	// for which references were discovered. This field is optional, and
	// only populated if the ImageSubscription specifies a Platform.
	Platform string `json:"platform,omitempty" protobuf:"bytes,2,opt,name=platform"`
	// References is a list of image references discovered by the Warehouse for
	// the ImageSubscription. An empty list indicates that the discovery
	// operation was successful, but no images matching the ImageSubscription
	// criteria were found.
	//
	// +optional
	References []DiscoveredImageReference `json:"references" protobuf:"bytes,3,rep,name=references"`
}

// DiscoveredImageReference represents an image reference discovered by a
// Warehouse for an ImageSubscription.
type DiscoveredImageReference struct {
	// Tag is the tag of the image.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^[\w.\-\_]+$`
	// +akuity:test-kubebuilder-pattern=Tag
	Tag string `json:"tag" protobuf:"bytes,1,opt,name=tag"`
	// Digest is the digest of the image.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[a-z0-9]+:[a-f0-9]+$`
	// +akuity:test-kubebuilder-pattern=Digest
	Digest string `json:"digest" protobuf:"bytes,2,opt,name=digest"`
	// Annotations is a map of key-value pairs that provide additional
	// information about the image.
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,5,rep,name=annotations" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// CreatedAt is the time the image was created. This field is optional, and
	// not populated for every ImageSelectionStrategy.
	CreatedAt *metav1.Time `json:"createdAt,omitempty" protobuf:"bytes,4,opt,name=createdAt"`
}

// ChartDiscoveryResult represents the result of a chart discovery operation for
// a ChartSubscription.
type ChartDiscoveryResult struct {
	// RepoURL is the repository URL of the Helm chart, as specified in the
	// ChartSubscription.
	//
	// +kubebuilder:validation:MinLength=1
	RepoURL string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	// Name is the name of the Helm chart, as specified in the ChartSubscription.
	Name string `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
	// SemverConstraint is the constraint for which versions were discovered.
	// This field is optional, and only populated if the ChartSubscription
	// specifies a SemverConstraint.
	SemverConstraint string `json:"semverConstraint,omitempty" protobuf:"bytes,3,opt,name=semverConstraint"`
	// Versions is a list of versions discovered by the Warehouse for the
	// ChartSubscription. An empty list indicates that the discovery operation was
	// successful, but no versions matching the ChartSubscription criteria were
	// found.
	//
	// +optional
	Versions []string `json:"versions" protobuf:"bytes,4,rep,name=versions"`
}

// DiscoveryResult represents the result of an artifact discovery operation for
// some subscription.
type DiscoveryResult struct {
	// SubscriptionName is the name of the Subscription that discovered these
	// results.
	//
	// +kubebuilder:validation:MinLength=1
	SubscriptionName string `json:"name" protobuf:"bytes,3,opt,name=name"`
	// ArtifactReferences is a list of references to specific versions of an
	// artifact.
	//
	// +optional
	ArtifactReferences []ArtifactReference `json:"artifactReferences" protobuf:"bytes,2,rep,name=artifactReferences"`
}

// ArtifactReference is a reference to a specific version of an artifact.
type ArtifactReference struct {
	// ArtifactType specifies the type of artifact this is. Often, but not always,
	// it will be the media type (MIME type) of the artifact referenced by this
	// ArtifactReference.
	//
	// +kubebuilder:validation:MinLength=1
	ArtifactType string `json:"artifactType,omitempty" protobuf:"bytes,1,opt,name=artifactType"`
	// SubscriptionName is the name of the Subscription that discovered this
	// artifact.
	//
	// +kubebuilder:validation:MinLength=1
	SubscriptionName string `json:"subscriptionName" protobuf:"bytes,2,opt,name=subscriptionName"`
	// Version identifies a specific revision of this artifact.
	//
	// +kubebuilder:validation:MinLength=1
	Version string `json:"version" protobuf:"bytes,3,opt,name=version"`
	// Metadata is a JSON object containing a mostly opaque collection of artifact
	// attributes. (It must be an object. It may not be a list or a scalar value.)
	// "Mostly" because Kargo may understand how to interpret some documented,
	// well-known, top-level keys. Those aside, this metadata is only understood
	// by a corresponding Subscriber implementation that created it.
	//
	// +optional
	Metadata *apiextensionsv1.JSON `json:"metadata,omitempty" protobuf:"bytes,4,opt,name=metadata"`
}

// DeepEquals returns a bool indicating whether the receiver deep-equals the
// provided ArtifactReference. I.e., all relevant fields must be equal.
func (g *ArtifactReference) DeepEquals(
	other *ArtifactReference,
) bool {
	if g == nil && other == nil {
		return true
	}
	if g == nil || other == nil {
		return false
	}
	if (g.Metadata == nil) != (other.Metadata == nil) {
		return false
	}
	return g.ArtifactType == other.ArtifactType &&
		g.SubscriptionName == other.SubscriptionName &&
		g.Version == other.Version &&
		// If we got to here and one Metadata is nil, the other must be nil too, so
		// we only need to look at one before knowing it's safe to compare Raw
		// values.
		(g.Metadata == nil || string(g.Metadata.Raw) == string(other.Metadata.Raw))
}

// +kubebuilder:object:root=true

// WarehouseList is a list of Warehouse resources.
type WarehouseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Warehouse `json:"items" protobuf:"bytes,2,rep,name=items"`
}
