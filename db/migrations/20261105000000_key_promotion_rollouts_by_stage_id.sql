-- +goose Up

-- A rollout belongs to one Stage's row, and so to one Stage's lifetime, by
-- the row's id. The rules and indexes that hold per Stage are keyed by that
-- id now, rather than by the Project and Stage names: numbers are unique
-- within the Stage's row, one rollout of the row may be Running, and a
-- Stage's lists read the row's rollouts. A reader that has only the names
-- finds the row's id through the stages table first.
--
-- The names stay on each rollout. A Project's rollouts are listed by
-- project_name, the API addresses a rollout by Project and rollout name, and
-- a row read whole carries its Stage's name. The Stage reference now
-- includes them, so the database holds them to the names of the Stage
-- whose id the rollout carries. A Stage's names never change for its id,
-- so the reference never needs to follow an update.

-- A foreign key must reference a unique key. The id alone is one, so this
-- one adds nothing but the reference's target.
ALTER TABLE stages
    ADD CONSTRAINT stages_id_project_name_name_key UNIQUE (id, project_name, name);

-- A Stage's counter starts at zero with its row. A Stage recreated before
-- rollouts referenced Stage rows kept its predecessor's rollouts, which the
-- reference then gave to the new row, numbered above the new row's counter.
-- Creating a rollout no longer looks past the counter, so the counter is
-- raised to the highest number its row's rollouts already have.
UPDATE stages SET last_promotion_rollout_number = numbered.number
FROM (
    SELECT stage_id, MAX(number) AS number
    FROM promotion_rollouts
    GROUP BY stage_id
) AS numbered
WHERE stages.id = numbered.stage_id
    AND stages.last_promotion_rollout_number < numbered.number;

-- The stage/number unique index serves a Stage's rollouts by number, in
-- either direction, and the cascade from the Stage's row.
ALTER TABLE promotion_rollouts
    DROP CONSTRAINT promotion_rollouts_stage_id_fkey,
    ADD CONSTRAINT promotion_rollouts_stage_fkey
        FOREIGN KEY (stage_id, project_name, stage_name)
        REFERENCES stages (id, project_name, name) ON DELETE CASCADE,
    DROP CONSTRAINT promotion_rollouts_stage_number_key,
    ADD CONSTRAINT promotion_rollouts_stage_number_key UNIQUE (stage_id, number);
DROP INDEX promotion_rollouts_stage_id_idx;

-- A Stage's rollouts of one piece of Freight, newest first.
DROP INDEX promotion_rollouts_freight_idx;
CREATE INDEX promotion_rollouts_freight_idx
    ON promotion_rollouts (stage_id, freight_name, number);

-- The rollouts still in progress, by Stage. The predicate must match the
-- queries' word for word.
DROP INDEX promotion_rollouts_open_idx;
CREATE INDEX promotion_rollouts_open_idx
    ON promotion_rollouts (stage_id, number)
    WHERE status_phase IN ('Pending', 'Running');

-- Two rollouts of one Stage cannot both be Running.
DROP INDEX promotion_rollouts_running_key;
CREATE UNIQUE INDEX promotion_rollouts_running_key
    ON promotion_rollouts (stage_id)
    WHERE status_phase = 'Running';

-- +goose Down
DROP INDEX promotion_rollouts_running_key;
CREATE UNIQUE INDEX promotion_rollouts_running_key
    ON promotion_rollouts (project_name, stage_name)
    WHERE status_phase = 'Running';
DROP INDEX promotion_rollouts_open_idx;
CREATE INDEX promotion_rollouts_open_idx
    ON promotion_rollouts (project_name, stage_name, number)
    WHERE status_phase IN ('Pending', 'Running');
DROP INDEX promotion_rollouts_freight_idx;
CREATE INDEX promotion_rollouts_freight_idx
    ON promotion_rollouts (project_name, stage_name, freight_name, number);
CREATE INDEX promotion_rollouts_stage_id_idx ON promotion_rollouts (stage_id);
ALTER TABLE promotion_rollouts
    DROP CONSTRAINT promotion_rollouts_stage_number_key,
    ADD CONSTRAINT promotion_rollouts_stage_number_key
        UNIQUE (project_name, stage_name, number),
    DROP CONSTRAINT promotion_rollouts_stage_fkey,
    ADD CONSTRAINT promotion_rollouts_stage_id_fkey
        FOREIGN KEY (stage_id) REFERENCES stages (id) ON DELETE CASCADE;
ALTER TABLE stages DROP CONSTRAINT stages_id_project_name_name_key;
