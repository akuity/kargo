-- +goose Up

-- A PromotionRollout's FreightCollection: one row per origin, holding the
-- Freight its Promotions render for that origin. A rollout promotes one
-- piece of Freight, but a Stage may request Freight from several origins,
-- and a Promotion renders all of it: the Freight being promoted and, for
-- each other origin, the Freight the Stage last promoted.
--
-- The rows are a snapshot, not a reference. reference is the whole
-- FreightReference as the API serves it -- the Freight's commits, images,
-- charts and artifacts with its name and origin -- because the record of
-- what a rollout promoted has to outlive the Freight itself, which is
-- garbage-collected in time. Nothing here references the Freight for that
-- reason. The contents are kept as one JSON value rather than in columns
-- and tables of their own: a collection is only ever read whole, and a
-- field added to the API type is kept without a migration. The origin and
-- the Freight's name are repeated as columns, as promotion_freight has
-- them, so that rows can be found by either without reading the JSON.
--
-- reference is json, not jsonb. jsonb rejects the escape \u0000, which a
-- string in the contents may hold -- an artifact's opaque metadata, say --
-- and a rollout given such Freight could then never start. json checks only
-- that the value is well formed and keeps it as written, which is all a
-- value that is only ever read whole needs. It has no equality operator, so
-- compare two references as text.
--
-- Rows are written once, with the write that moves the rollout from Pending
-- to Running, and never updated. A rollout started before this table
-- existed has none. They go with their rollout, which goes with its Stage.
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
