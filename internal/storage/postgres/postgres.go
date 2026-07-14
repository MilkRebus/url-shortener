package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/milkrebus/url-shortener/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type database interface {
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Ping(ctx context.Context) error
	Close()
}

type Storage struct {
	pool database
}

func New(ctx context.Context, databaseURL string, maxConns, minConns int32) (*Storage, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL config: %w", err)
	}
	if maxConns > 0 {
		config.MaxConns = maxConns
	}
	if minConns >= 0 {
		config.MinConns = minConns
	}
	config.MaxConnIdleTime = 5 * time.Minute
	config.MaxConnLifetime = 30 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return &Storage{pool: pool}, nil
}

func (s *Storage) CreateOrGet(ctx context.Context, originalURL, proposedCode string) (storage.Link, bool, error) {
	var link storage.Link
	err := s.pool.QueryRow(ctx, `
		INSERT INTO links (code, original_url)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
		RETURNING code, original_url, created_at
	`, proposedCode, originalURL).Scan(&link.Code, &link.OriginalURL, &link.CreatedAt)
	if err == nil {
		return link, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return storage.Link{}, false, fmt.Errorf("insert link: %w", err)
	}
	err = s.pool.QueryRow(ctx, `
		SELECT code, original_url, created_at
		FROM links
		WHERE original_url = $1
	`, originalURL).Scan(&link.Code, &link.OriginalURL, &link.CreatedAt)
	if err == nil {
		return link, false, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return storage.Link{}, false, storage.ErrCodeCollision
	}
	return storage.Link{}, false, fmt.Errorf("get existing link: %w", err)
}

func (s *Storage) GetByCode(ctx context.Context, code string) (storage.Link, error) {
	var link storage.Link
	err := s.pool.QueryRow(ctx, `
		SELECT code, original_url, created_at
		FROM links
		WHERE code = $1
	`, code).Scan(&link.Code, &link.OriginalURL, &link.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return storage.Link{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.Link{}, fmt.Errorf("get link by code: %w", err)
	}
	return link, nil
}

func (s *Storage) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Storage) Close() {
	s.pool.Close()
}
