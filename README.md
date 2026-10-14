# PRismaGo

Backend API for a weightlifting PR tracking and analytics dashboard. Go + PostgreSQL ([pgx](https://github.com/jackc/pgx), [goose](https://github.com/pressly/goose) migrations).

## Run

```sh
docker compose up -d --wait   # Postgres 17 on :5432
go run ./cmd/api              # applies migrations, serves on :8080
go test ./...                 # unit tests; Postgres test is skipped
TEST_DATABASE_URL=postgres://prismago:prismago@localhost:5432/prismago?sslmode=disable go test ./...
```

Env vars: `PORT`, `LOG_LEVEL`, `DATABASE_URL` (defaults to the docker compose database).
`docker compose down` stops Postgres and keeps data; `down -v` wipes it.

New schema change: add `internal/postgres/migrations/0000N_description.sql` with `-- +goose Up` / `-- +goose Down` sections. It runs on next startup.

```sh
curl localhost:8080/healthz
# register (height/weight in the chosen units); the response contains a token
curl -X POST localhost:8080/api/v1/auth/register -d '{"email":"sam@example.com","name":"Sam","sex":"male","units":"imperial","height":71,"weight":190,"password":"correct horse"}'
curl -X POST localhost:8080/api/v1/me/lifts -H "Authorization: Bearer <TOKEN>" -d '{"exercise":"squat","weight_kg":140,"reps":3}'
curl "localhost:8080/api/v1/me/lifts?exercise=squat" -H "Authorization: Bearer <TOKEN>"
```

| Method | Path | Auth | |
|---|---|---|---|
| POST | `/api/v1/auth/register` | — | profile + `password` (8-72 bytes); returns `{token, expires_at, user}`; 409 if email taken |
| POST | `/api/v1/auth/login` | — | `{email, password}`; same response; 401 on any mismatch |
| POST | `/api/v1/auth/logout` | token | ends that session only; always 204 |
| GET, PUT, DELETE | `/api/v1/me` | token | own profile; DELETE removes account, lifts and sessions |
| GET, POST | `/api/v1/me/lifts` | token | list (`?exercise=`) / create |
| GET, DELETE | `/api/v1/me/lifts/{id}` | token | 404 for another user's lift |

Tokens last 30 days and are sent as `Authorization: Bearer <token>`. Missing or invalid tokens get `401` with `WWW-Authenticate: Bearer`.

## Layout

```
cmd/api/            entrypoint: config + dependency wiring only
internal/
  config/           env-based settings
  server/           http.Server, mux, middleware, graceful shutdown
  httpx/            JSON encode/decode/error helpers
  postgres/         connection pool + embedded migrations
  session/          bearer tokens, sessions table, RequireUser middleware
  auth/             register/login/logout, bcrypt (no table of its own)
  user/             user profiles (same layout as lift/)
  lift/             feature package (one per domain area)
    model.go        types, domain errors, pure logic
    repository.go   storage interface + in-memory impl (used by unit tests)
    postgres.go     Postgres implementation of the interface
    service.go      business rules
    handler.go      HTTP adapter
```

Add a feature: create `internal/<feature>/` with the same four files, then wire it in `cmd/api/main.go`.
