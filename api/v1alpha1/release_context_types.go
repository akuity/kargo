package v1alpha1

// ReleaseContextConfig controls how the dashboard interprets image annotations
// already stored in Freight. Standard OCI annotations retain their meanings.
type ReleaseContextConfig struct {
	// ImageAnnotations maps optional commit details to exact custom annotation
	// keys. Unconfigured or absent values are not interpreted.
	// +optional
	ImageAnnotations ImageAnnotationMappings `json:"imageAnnotations,omitempty"`
}

// ImageAnnotationKey identifies a custom image annotation, not a key in the
// reserved org.opencontainers namespace.
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:Pattern=`^\S+$`
// +kubebuilder:validation:XValidation:rule="!self.startsWith('org.opencontainers.')",message="must be a custom annotation key"
type ImageAnnotationKey string

// ImageAnnotationMappings selects the custom annotations to display as commit
// details. These mappings do not affect image discovery or Freight storage.
type ImageAnnotationMappings struct {
	// CommitSubject is the annotation containing the source commit's subject.
	// +optional
	CommitSubject ImageAnnotationKey `json:"commitSubject,omitempty"`
	// CommitAuthor is the annotation containing the source commit's author.
	// +optional
	CommitAuthor ImageAnnotationKey `json:"commitAuthor,omitempty"`
	// CommitCommitter is the annotation containing the source commit's committer.
	// +optional
	CommitCommitter ImageAnnotationKey `json:"commitCommitter,omitempty"`
	// CommitCreatedAt is the annotation containing the source commit timestamp
	// in RFC 3339 format. It is separate from the OCI image creation timestamp.
	// +optional
	CommitCreatedAt ImageAnnotationKey `json:"commitCreatedAt,omitempty"`
}
