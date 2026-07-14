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

	"github.com/milkrebus/url-shortener/internal/config"
	"github.com/milkrebus/url-shortener/internal/generator"
	"github.com/milkrebus/url-shortener/internal/service"
	"github.com/milkrebus/url-shortener/internal/storage"
	"github.com/milkrebus/url-shortener/internal/storage/memory"
	postgresstorage "github.com/milkrebus/url-shortener/internal/storage/postgres"
	"github.com/milkrebus/url-shortener/internal/transport/httpapi"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("application stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	store, err := newStorage(ctx, cfg)
	if err != nil {
		return fmt.Errorf("initialize storage: %w", err)
	}
	defer store.Close()
	linkService, err := service.New(store, generator.NewRandom(), cfg.BaseURL, cfg.GenerationAttempts)
	if err != nil {
		return fmt.Errorf("initialize service: %w", err)
	}
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.New(linkService, store.Ping, logger),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server started", "address", cfg.Address, "storage", cfg.StorageType)
		serverErrors <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("HTTP server stopped")
	return nil
}

func newStorage(ctx context.Context, cfg config.Config) (storage.Storage, error) {
	if cfg.StorageType == "memory" {
		return memory.New(), nil
	}
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return postgresstorage.New(connectCtx, cfg.DatabaseURL, int32(cfg.DBMaxConns), int32(cfg.DBMinConns))
}
