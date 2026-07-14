package storage

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("link not found")
	ErrCodeCollision = errors.New("short code collision")
)

type Link struct {
	Code        string
	OriginalURL string
	CreatedAt   time.Time
}

type Storage interface {
	CreateOrGet(ctx context.Context, originalURL, proposedCode string) (Link, bool, error)
	GetByCode(ctx context.Context, code string) (Link, error)
	Ping(ctx context.Context) error
	Close()
}
