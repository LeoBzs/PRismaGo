-- +goose Up
-- Every lift belongs to a user; deleting the user deletes their lifts.
-- Adding NOT NULL directly only works because lifts was empty when this was
-- written. On a table with data: add nullable, backfill, then SET NOT NULL.
ALTER TABLE lifts
    ADD COLUMN user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE;

-- Every lift query is now scoped to one user, so user_id leads the index.
-- It also serves the foreign key (cascade deletes look up lifts by user_id).
DROP INDEX lifts_exercise_performed_at_idx;
CREATE INDEX lifts_user_exercise_performed_at_idx ON lifts (user_id, exercise, performed_at DESC);

-- +goose Down
DROP INDEX lifts_user_exercise_performed_at_idx;
CREATE INDEX lifts_exercise_performed_at_idx ON lifts (exercise, performed_at DESC);
ALTER TABLE lifts DROP COLUMN user_id;
