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
	// FreightCreatedKey is the attribute under which whether a create call
	// made a new Freight is recorded.
	FreightCreatedKey = attribute.Key("kargo.freight.created")

	// KindKey is the attribute under which the kind of a Kubernetes object a
	// span concerns is recorded.
	KindKey = attribute.Key("kargo.object.kind")
	// NameKey is the attribute under which the name of the Kubernetes object
	// a span concerns is recorded.
	NameKey = attribute.Key("kargo.object.name")
	// NamespaceKey is the attribute under which the namespace of a Kubernetes
	// object that is not a Kargo object is recorded (OTel semconv).
	NamespaceKey = semconv.K8SNamespaceNameKey

	// SubscriptionsKey is the attribute under which the number of a
	// Warehouse's subscriptions is recorded.
	SubscriptionsKey = attribute.Key("kargo.subscriptions")
	// SubscriptionKindKey is the attribute under which a subscription's kind
	// (git, image, chart, other) is recorded.
	SubscriptionKindKey = attribute.Key("kargo.subscription.kind")
	// SubscriptionRepoKey is the attribute under which a subscription's
	// repository URL, without userinfo or query, is recorded.
	SubscriptionRepoKey = attribute.Key("kargo.subscription.repo_url")
	// DiscoveredKey is the attribute under which the number of commits, image
	// references or chart versions a subscription discovered is recorded.
	DiscoveredKey = attribute.Key("kargo.discovery.count")
	// DiscoveredNewestKey is the attribute under which the newest commit ID,
	// image tag or chart version a subscription discovered is recorded.
	DiscoveredNewestKey = attribute.Key("kargo.discovery.newest")
	// CloneSkippedKey is the attribute under which whether Git discovery
	// skipped its clone is recorded.
	CloneSkippedKey = attribute.Key("kargo.discovery.clone_skipped")

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
