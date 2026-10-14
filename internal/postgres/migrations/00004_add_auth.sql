-- +goose Up
-- bcrypt hash; the plaintext password is never stored.
-- NOT NULL directly only because users was empty when this was written.
ALTER TABLE users ADD COLUMN password_hash text NOT NULL;

-- One row per logged-in device. token_hash is SHA-256 of the bearer token, so
-- a leaked table can't be replayed as logins. Deleting a user ends their sessions.
CREATE TABLE sessions (
    token_hash bytea       PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

-- +goose Down
DROP TABLE sessions;
ALTER TABLE users DROP COLUMN password_hash;
