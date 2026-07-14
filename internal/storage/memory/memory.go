package memory

import (
	"context"
	"sync"
	"time"

	"github.com/milkrebus/url-shortener/internal/storage"
)

type Storage struct {
	mu     sync.RWMutex
	byCode map[string]storage.Link
	byURL  map[string]string
}

func New() *Storage {
	return &Storage{
		byCode: make(map[string]storage.Link),
		byURL:  make(map[string]string),
	}
}

func (s *Storage) CreateOrGet(_ context.Context, originalURL, proposedCode string) (storage.Link, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if code, ok := s.byURL[originalURL]; ok {
		return s.byCode[code], false, nil
	}
	if _, ok := s.byCode[proposedCode]; ok {
		return storage.Link{}, false, storage.ErrCodeCollision
	}
	link := storage.Link{
		Code:        proposedCode,
		OriginalURL: originalURL,
		CreatedAt:   time.Now().UTC(),
	}
	s.byCode[proposedCode] = link
	s.byURL[originalURL] = proposedCode
	return link, true, nil
}

func (s *Storage) GetByCode(_ context.Context, code string) (storage.Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	link, ok := s.byCode[code]
	if !ok {
		return storage.Link{}, storage.ErrNotFound
	}
	return link, nil
}

func (s *Storage) Ping(context.Context) error {
	return nil
}

func (s *Storage) Close() {}
