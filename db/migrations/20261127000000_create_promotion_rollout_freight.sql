-- +goose Up

-- A PromotionRollout's FreightCollection: the Freight its Promotions
-- render, one row per origin. That is the Freight being promoted plus, for
-- each other origin the Stage requests Freight from, the Freight carried
-- forward from the Stage's previous rollout.
--
-- Rows are written once, when the rollout starts, and never updated. A
-- rollout started before this table existed has none.
--
-- Columns:
--   * reference: a full copy of the Freight, as a serialized
--     FreightReference (its name, origin, commits, images, charts and
--     artifacts). It is a copy, with no foreign key to the Freight, because
--     Freight is garbage-collected and this record must outlive it.
--   * origin_kind, origin_name, freight_name: repeated from reference so
--     that rows can be found without reading the JSON.
--
-- reference is json rather than jsonb because jsonb rejects the \u0000
-- escape, which Freight contents such as artifact metadata may hold. json
-- has no equality operator, so compare references as text.
CREATE TABLE promotion_rollout_freight (
    promotion_rollout_id UUID NOT NULL REFERENCES promotion_rollouts (id) ON DELETE CASCADE,
    origin_kind TEXT NOT NULL CHECK (origin_kind <> ''),
    origin_name TEXT NOT NULL CHECK (origin_name <> ''),
    freight_name TEXT NOT NULL CHECK (freight_name <> ''),
    reference JSON NOT NULL CHECK (json_typeof(reference) = 'object'),
    PRIMARY KEY (promotion_rollout_id, origin_kind, origin_name)
);

-- +goose Down
DROP TABLE promotion_rollout_freight;
