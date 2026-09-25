package telemetry

import "go.opentelemetry.io/otel/attribute"

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
