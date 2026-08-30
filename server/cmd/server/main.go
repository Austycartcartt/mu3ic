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
	"strings"
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

// mustEnv returns the value of key or exits if it's unset/empty. Used for
// secrets and backend config that have no safe default in production.
func mustEnv(logger *slog.Logger, key string) string {
	v := os.Getenv(key)
	if v == "" {
		logger.Error("required environment variable is not set", "var", key)
		os.Exit(1)
	}
	return v
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	port := getEnv("PORT", "8080")
	databaseURL := getEnv("DATABASE_URL", "postgres://mu3ic:mu3ic@localhost:5432/mu3ic?sslmode=disable")
	dataDir := getEnv("DATA_DIR", "./data")
	migrationsDir := getEnv("MIGRATIONS_DIR", "./migrations")

	// JWT_SECRET has no safe default: an unset or well-known value lets
	// anyone mint valid tokens. Fail fast rather than warn and run.
	const devJWTSecret = "dev-secret-change-in-production"
	jwtSecret := os.Getenv("JWT_SECRET")
	switch {
	case jwtSecret == "", jwtSecret == devJWTSecret:
		logger.Error("JWT_SECRET must be set to a strong random value (e.g. `openssl rand -hex 32`)")
		os.Exit(1)
	case len(jwtSecret) < 32:
		logger.Error("JWT_SECRET is too short; use at least 32 characters")
		os.Exit(1)
	}

	storage := buildStorage(logger, dataDir)

	registration := api.RegistrationPolicy{
		Open:       getEnv("REGISTRATION_OPEN", "false") == "true",
		InviteCode: os.Getenv("REGISTRATION_INVITE_CODE"),
	}
	trustProxy := getEnv("TRUST_PROXY", "false") == "true"
	streamURLTTL := parseDurEnv(logger, "STREAM_URL_TTL", 15*time.Minute)

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

	// Upload staging is always local: the multipart body is streamed to a
	// real file so dhowden/tag has an io.ReadSeeker, then handed to
	// Storage.Put. This is needed even when Storage is a remote object
	// store.
	cfg := library.Config{LibraryDir: dataDir}
	if err := os.MkdirAll(cfg.TempDir(), 0o755); err != nil {
		logger.Error("creating temp dir", "error", err)
		os.Exit(1)
	}

	srv := api.NewServer(api.Options{
		Store:        store.New(db),
		Storage:      storage,
		Config:       cfg,
		Logger:       logger,
		JWTSecret:    jwtSecret,
		Registration: registration,
		TrustProxy:   trustProxy,
		StreamURLTTL: streamURLTTL,
	})

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

// buildStorage selects the audio/artwork backend from STORAGE_BACKEND:
// "fs" (default) keeps files under dataDir; "r2" stores them in a
// Cloudflare R2 bucket; "neon" stores them in a Neon Object Storage
// bucket. Both object backends serve via short-lived presigned URLs.
func buildStorage(logger *slog.Logger, dataDir string) library.Storage {
	switch backend := getEnv("STORAGE_BACKEND", "fs"); backend {
	case "fs":
		s, err := library.NewFileStorage(dataDir)
		if err != nil {
			logger.Error("initializing filesystem storage", "error", err)
			os.Exit(1)
		}
		return s
	case "r2":
		s, err := library.NewR2Storage(library.R2Config{
			Endpoint:        mustEnv(logger, "R2_ENDPOINT"),
			Bucket:          mustEnv(logger, "R2_BUCKET"),
			AccessKeyID:     mustEnv(logger, "R2_ACCESS_KEY_ID"),
			SecretAccessKey: mustEnv(logger, "R2_SECRET_ACCESS_KEY"),
		})
		if err != nil {
			logger.Error("initializing R2 storage", "error", err)
			os.Exit(1)
		}
		return s
	case "neon":
		// Endpoint / region / credentials use the AWS_* names Neon
		// injects (see `neon env pull`), so pulled vars drop straight
		// into the environment. Neon does not inject the bucket name, so
		// NEON_STORAGE_BUCKET carries it. AWS_REGION is optional —
		// NewNeonStorage defaults it to the beta's only region.
		s, err := library.NewNeonStorage(library.NeonConfig{
			Endpoint:        mustEnv(logger, "AWS_ENDPOINT_URL_S3"),
			Bucket:          mustEnv(logger, "NEON_STORAGE_BUCKET"),
			AccessKeyID:     mustEnv(logger, "AWS_ACCESS_KEY_ID"),
			SecretAccessKey: mustEnv(logger, "AWS_SECRET_ACCESS_KEY"),
			Region:          os.Getenv("AWS_REGION"),
		})
		if err != nil {
			logger.Error("initializing Neon storage", "error", err)
			os.Exit(1)
		}
		return s
	default:
		logger.Error("invalid STORAGE_BACKEND (want \"fs\", \"r2\", or \"neon\")", "value", backend)
		os.Exit(1)
		return nil // unreachable
	}
}

// parseDurEnv reads a time.Duration from key, falling back to def when
// unset and exiting on an unparseable value.
func parseDurEnv(logger *slog.Logger, key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		logger.Error("invalid duration in environment variable", "var", key, "value", v, "error", err)
		os.Exit(1)
	}
	return d
}
