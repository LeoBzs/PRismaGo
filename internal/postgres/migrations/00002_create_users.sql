-- +goose Up
-- Body measurements are canonical metric; units is only the display preference.
CREATE TABLE users (
    id             uuid          PRIMARY KEY,
    email          text          NOT NULL UNIQUE CHECK (email <> ''),
    name           text          NOT NULL CHECK (name <> ''),
    sex            text          NOT NULL CHECK (sex IN ('male', 'female')),
    units          text          NOT NULL CHECK (units IN ('metric', 'imperial')),
    height_cm      numeric(5, 2) NOT NULL CHECK (height_cm BETWEEN 100 AND 250),
    body_weight_kg numeric(5, 2) NOT NULL CHECK (body_weight_kg BETWEEN 30 AND 300),
    created_at     timestamptz   NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
