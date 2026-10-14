// Command api is the HTTP entrypoint. Its only job is wiring: load config,
// build dependencies, hand them to the server. No business logic lives here.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/LeoBzs/prismago/internal/auth"
	"github.com/LeoBzs/prismago/internal/config"
	"github.com/LeoBzs/prismago/internal/lift"
	"github.com/LeoBzs/prismago/internal/postgres"
	"github.com/LeoBzs/prismago/internal/server"
	"github.com/LeoBzs/prismago/internal/session"
	"github.com/LeoBzs/prismago/internal/user"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Database: connect, then bring the schema up to date before serving.
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool, logger); err != nil {
		return err
	}

	// Feature wiring: repository -> service -> handler.
	// Sessions come first: every protected handler takes its RequireUser middleware.
	sessionService := session.NewService(session.NewPostgresRepository(pool))
	requireUser := sessionService.RequireUser

	userService := user.NewService(user.NewPostgresRepository(pool))
	userHandler := user.NewHandler(userService, requireUser)

	liftService := lift.NewService(lift.NewPostgresRepository(pool))
	liftHandler := lift.NewHandler(liftService, requireUser)

	// auth composes the other two: users for credentials, sessions for tokens.
	authHandler := auth.NewHandler(auth.NewService(userService, sessionService))

	srv := server.New(cfg, logger, authHandler, userHandler, liftHandler)

	return srv.Run(ctx)
}
