package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/milkrebus/url-shortener/internal/generator"
	"github.com/milkrebus/url-shortener/internal/storage"
)

var (
	ErrInvalidURL          = errors.New("invalid URL")
	ErrInvalidCode         = errors.New("invalid short code")
	ErrGenerationExhausted = errors.New("could not generate a unique short code")
)

type CreateResult struct {
	Link     storage.Link
	ShortURL string
	Created  bool
}

type Service struct {
	storage     storage.Storage
	generator   generator.Generator
	baseURL     string
	maxAttempts int
}

func New(store storage.Storage, codeGenerator generator.Generator, baseURL string, maxAttempts int) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("storage is nil")
	}
	if codeGenerator == nil {
		return nil, fmt.Errorf("generator is nil")
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("invalid base URL %q", baseURL)
	}
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	return &Service{
		storage:     store,
		generator:   codeGenerator,
		baseURL:     baseURL,
		maxAttempts: maxAttempts,
	}, nil
}

func (s *Service) Create(ctx context.Context, rawURL string) (CreateResult, error) {
	originalURL, err := validateURL(rawURL)
	if err != nil {
		return CreateResult{}, err
	}
	for attempt := 0; attempt < s.maxAttempts; attempt++ {
		code, err := s.generator.Generate()
		if err != nil {
			return CreateResult{}, fmt.Errorf("generate short code: %w", err)
		}
		if !generator.IsValidCode(code) {
			return CreateResult{}, fmt.Errorf("generator returned invalid code")
		}
		link, created, err := s.storage.CreateOrGet(ctx, originalURL, code)
		if err == nil {
			return CreateResult{
				Link:     link,
				ShortURL: s.baseURL + "/" + link.Code,
				Created:  created,
			}, nil
		}
		if errors.Is(err, storage.ErrCodeCollision) {
			continue
		}
		return CreateResult{}, fmt.Errorf("store link: %w", err)
	}
	return CreateResult{}, ErrGenerationExhausted
}

func (s *Service) Get(ctx context.Context, code string) (storage.Link, error) {
	if !generator.IsValidCode(code) {
		return storage.Link{}, ErrInvalidCode
	}
	link, err := s.storage.GetByCode(ctx, code)
	if err != nil {
		return storage.Link{}, err
	}
	return link, nil
}

func validateURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || len(rawURL) > 8192 {
		return "", ErrInvalidURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil ||
		!parsed.IsAbs() ||
		parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", ErrInvalidURL
	}

	if parsed.Path == "" {
		parsed.Path = "/"
	}

	return parsed.String(), nil
}
