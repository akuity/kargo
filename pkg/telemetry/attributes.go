package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Attribute keys shared by Kargo's spans. Keeping them here, rather than
// spelled out at each call site, keeps the attributes consistent so that they
// can be queried across components.
const (
	// ProjectKey is the attribute under which the name of the Kargo Project
	// (i.e. Kubernetes namespace) a span concerns is recorded.
	ProjectKey = attribute.Key("kargo.project")
	// ShardKey is the attribute under which the name of a controller's shard
	// is recorded.
	ShardKey = attribute.Key("kargo.shard")
	// DefaultControllerKey is the attribute under which whether a controller
	// is the default controller, i.e. the one responsible for resources not
	// assigned to a specific shard, is recorded. A controller with no shard
	// name is always the default controller; one with a shard name is the
	// default controller only if explicitly designated as such.
	DefaultControllerKey = attribute.Key("kargo.controller.default")
	// StageKey is the attribute under which the name of a Stage is recorded.
	StageKey = attribute.Key("kargo.stage")
	// WarehouseKey is the attribute under which the name of a Warehouse is
	// recorded.
	WarehouseKey = attribute.Key("kargo.warehouse")
	// PromotionKey is the attribute under which the name of a Promotion is
	// recorded.
	PromotionKey = attribute.Key("kargo.promotion")
	// PromotionRequestKey is the attribute under which the name of a
	// PromotionRequest is recorded.
	PromotionRequestKey = attribute.Key("kargo.promotion_request")
	// FreightKey is the attribute under which the name of a Freight is
	// recorded.
	FreightKey = attribute.Key("kargo.freight")
	// FreightOriginKey is the attribute under which a Freight's origin is
	// recorded as kind/name, e.g. "Warehouse/smoke".
	FreightOriginKey = attribute.Key("kargo.freight.origin")
	// VCSCommitKey is the attribute under which a Git commit ID is recorded
	// (OTel semconv).
	VCSCommitKey = semconv.VCSRefHeadRevisionKey

	// NameKey is the attribute under which the name of the Kubernetes object
	// a span concerns is recorded.
	NameKey = attribute.Key("kargo.object.name")
	// NamespaceKey is the attribute under which the namespace of a Kubernetes
	// object that is not a Kargo object is recorded (OTel semconv).
	NamespaceKey = semconv.K8SNamespaceNameKey
	// ResourceKey is the attribute under which a Kubernetes API request's
	// resource (e.g. "freights") is recorded.
	ResourceKey = attribute.Key("kargo.k8s.resource")
	// SubresourceKey is the attribute under which a Kubernetes API request's
	// subresource (e.g. "status") is recorded.
	SubresourceKey = attribute.Key("kargo.k8s.subresource")
	// VerbKey is the attribute under which a Kubernetes API request's verb
	// (e.g. "patch") is recorded.
	VerbKey = attribute.Key("kargo.k8s.verb")

	// StepAliasKey is the attribute under which a promotion step's alias is
	// recorded.
	StepAliasKey = attribute.Key("kargo.promotion.step.alias")
	// StepKindKey is the attribute under which a promotion step's kind is
	// recorded, e.g. "git-clone".
	StepKindKey = attribute.Key("kargo.promotion.step.kind")
	// StepStatusKey is the attribute under which the status a promotion step
	// finished in is recorded.
	StepStatusKey = attribute.Key("kargo.promotion.step.status")
)
