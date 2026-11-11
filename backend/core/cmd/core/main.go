package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	corehttp "github.com/TSX97/AI-business-support/backend/core/internal/http"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

const defaultPort = "8081"

func main() {
	if err := run(); err != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		logger.Error("core service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return err
	}
	redisURL, err := requiredEnv("REDIS_URL")
	if err != nil {
		return err
	}
	natsURL, err := requiredEnv("NATS_URL")
	if err != nil {
		return err
	}

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStartup()

	postgres, err := pgxpool.New(startupCtx, databaseURL)
	if err != nil {
		return fmt.Errorf("create PostgreSQL connection pool: %w", err)
	}
	defer postgres.Close()
	if err := postgres.Ping(startupCtx); err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	redisOptions, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("parse REDIS_URL: %w", err)
	}
	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close()
	if err := redisClient.Ping(startupCtx).Err(); err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}

	natsConnection, err := nats.Connect(
		natsURL,
		nats.Name("core-service"),
		nats.Timeout(5*time.Second),
	)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	defer natsConnection.Close()

	server := &http.Server{
		Addr: ":" + port,
		Handler: corehttp.NewRouter(corehttp.Dependencies{
			Postgres: postgres,
			Redis:    redisClient,
			NATS:     natsConnection,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("starting core service", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("core service stopped unexpectedly: %w", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		logger.Info("core service stopped")
	}

	return nil
}

func requiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("%s environment variable is required", name)
	}
	return value, nil
}
