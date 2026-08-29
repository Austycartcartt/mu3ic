// Command server is the mu3ic API entry point: wiring, config, and
// graceful shutdown live here so the rest of the packages stay
// framework-agnostic.
package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver

	"github.com/Austycartcartt/mu3ic/server/internal/api"
	"github.com/Austycartcartt/mu3ic/server/internal/library"
	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	port := getEnv("PORT", "8080")
	databaseURL := getEnv("DATABASE_URL", "postgres://mu3ic:mu3ic@localhost:5432/mu3ic?sslmode=disable")
	dataDir := getEnv("DATA_DIR", "./data")
	migrationsDir := getEnv("MIGRATIONS_DIR", "./migrations")

	const devJWTSecret = "dev-secret-change-in-production"
	jwtSecret := getEnv("JWT_SECRET", devJWTSecret)
	if jwtSecret == devJWTSecret {
		logger.Warn("JWT_SECRET not set, using an insecure development default; set JWT_SECRET before deploying")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		logger.Error("opening database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		logger.Error("connecting to database", "error", err)
		os.Exit(1)
	}

	if err := store.RunMigrations(ctx, db, migrationsDir); err != nil {
		logger.Error("running migrations", "error", err)
		os.Exit(1)
	}

	storage, err := library.NewFileStorage(dataDir)
	if err != nil {
		logger.Error("initializing storage", "error", err)
		os.Exit(1)
	}

	cfg := library.Config{LibraryDir: dataDir}
	if err := os.MkdirAll(cfg.TempDir(), 0o755); err != nil {
		logger.Error("creating temp dir", "error", err)
		os.Exit(1)
	}

	srv := api.NewServer(store.New(db), storage, cfg, logger, jwtSecret)

	httpServer := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: srv.Router(),
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting server", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownCtx.Done()
	logger.Info("shutting down")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(timeoutCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}
