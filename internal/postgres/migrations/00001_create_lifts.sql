-- +goose Up
CREATE TABLE lifts (
    id           uuid          PRIMARY KEY,
    exercise     text          NOT NULL CHECK (exercise <> ''),
    weight_kg    numeric(6, 2) NOT NULL CHECK (weight_kg > 0),
    reps         integer       NOT NULL CHECK (reps > 0),
    performed_at timestamptz   NOT NULL,
    created_at   timestamptz   NOT NULL DEFAULT now()
);

-- Serves "history for one exercise, newest first" (list filter + future PR queries).
CREATE INDEX lifts_exercise_performed_at_idx ON lifts (exercise, performed_at DESC);

-- +goose Down
DROP TABLE lifts;
