package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address            string
	BaseURL            string
	StorageType        string
	DatabaseURL        string
	DBMaxConns         int
	DBMinConns         int
	GenerationAttempts int
	ReadHeaderTimeout  time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	ShutdownTimeout    time.Duration
}

func Load(args []string) (Config, error) {
	dbMaxConns, err := envInt("DB_MAX_CONNS", 10)
	if err != nil {
		return Config{}, err
	}
	dbMinConns, err := envInt("DB_MIN_CONNS", 1)
	if err != nil {
		return Config{}, err
	}
	generationAttempts, err := envInt("GENERATION_ATTEMPTS", 10)
	if err != nil {
		return Config{}, err
	}
	readHeaderTimeout, err := envDuration("READ_HEADER_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	readTimeout, err := envDuration("READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	writeTimeout, err := envDuration("WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	idleTimeout, err := envDuration("IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := envDuration("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg := Config{}
	fs := flag.NewFlagSet("url-shortener", flag.ContinueOnError)
	fs.StringVar(&cfg.Address, "addr", env("APP_ADDR", ":8080"), "HTTP server address")
	fs.StringVar(&cfg.BaseURL, "base-url", env("BASE_URL", "http://localhost:8080"), "public base URL")
	fs.StringVar(&cfg.StorageType, "storage", env("STORAGE_TYPE", "memory"), "storage type: memory or postgres")
	fs.StringVar(&cfg.DatabaseURL, "database-url", env("DATABASE_URL", ""), "PostgreSQL connection URL")
	fs.IntVar(&cfg.DBMaxConns, "db-max-conns", dbMaxConns, "maximum PostgreSQL connections")
	fs.IntVar(&cfg.DBMinConns, "db-min-conns", dbMinConns, "minimum PostgreSQL connections")
	fs.IntVar(&cfg.GenerationAttempts, "generation-attempts", generationAttempts, "maximum code generation attempts")
	fs.DurationVar(&cfg.ReadHeaderTimeout, "read-header-timeout", readHeaderTimeout, "HTTP read header timeout")
	fs.DurationVar(&cfg.ReadTimeout, "read-timeout", readTimeout, "HTTP read timeout")
	fs.DurationVar(&cfg.WriteTimeout, "write-timeout", writeTimeout, "HTTP write timeout")
	fs.DurationVar(&cfg.IdleTimeout, "idle-timeout", idleTimeout, "HTTP idle timeout")
	fs.DurationVar(&cfg.ShutdownTimeout, "shutdown-timeout", shutdownTimeout, "graceful shutdown timeout")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	cfg.StorageType = strings.ToLower(strings.TrimSpace(cfg.StorageType))
	if cfg.StorageType != "memory" && cfg.StorageType != "postgres" {
		return Config{}, fmt.Errorf("unsupported storage type %q", cfg.StorageType)
	}
	if cfg.StorageType == "postgres" && strings.TrimSpace(cfg.DatabaseURL) == "" {
		return Config{}, fmt.Errorf("database URL is required for postgres storage")
	}
	if cfg.DBMinConns < 0 || cfg.DBMaxConns <= 0 || cfg.DBMinConns > cfg.DBMaxConns {
		return Config{}, fmt.Errorf("invalid PostgreSQL pool limits")
	}
	if cfg.GenerationAttempts <= 0 {
		return Config{}, fmt.Errorf("generation attempts must be positive")
	}
	return cfg, nil
}

func env(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) (int, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func envDuration(name string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}
