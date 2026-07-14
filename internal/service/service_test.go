package service

import (
	"context"
	"errors"
	"testing"

	"github.com/milkrebus/url-shortener/internal/storage/memory"
)

type sequenceGenerator struct {
	codes []string
	err   error
	index int
}

func (g *sequenceGenerator) Generate() (string, error) {
	if g.err != nil {
		return "", g.err
	}
	if g.index >= len(g.codes) {
		return g.codes[len(g.codes)-1], nil
	}
	code := g.codes[g.index]
	g.index++
	return code, nil
}

func TestCreateAndGet(t *testing.T) {
	store := memory.New()
	svc, err := New(store, &sequenceGenerator{codes: []string{"aB3_q9ZxK2"}}, "http://localhost:8080", 10)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	result, err := svc.Create(context.Background(), "https://example.com/path")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !result.Created {
		t.Fatal("Create() created = false, want true")
	}
	if result.ShortURL != "http://localhost:8080/aB3_q9ZxK2" {
		t.Fatalf("Create() short URL = %q", result.ShortURL)
	}
	link, err := svc.Get(context.Background(), result.Link.Code)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if link.OriginalURL != "https://example.com/path" {
		t.Fatalf("Get() URL = %q", link.OriginalURL)
	}
}

func TestCreateSameURLReturnsSameCode(t *testing.T) {
	store := memory.New()
	generator := &sequenceGenerator{codes: []string{"aaaaaaaaaa", "bbbbbbbbbb"}}
	svc, err := New(store, generator, "http://localhost:8080", 10)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	first, err := svc.Create(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	second, err := svc.Create(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if second.Created {
		t.Fatal("second Create() created = true, want false")
	}
	if second.Link.Code != first.Link.Code {
		t.Fatalf("second code = %q, want %q", second.Link.Code, first.Link.Code)
	}
}

func TestCreateRetriesAfterCollision(t *testing.T) {
	store := memory.New()
	_, _, err := store.CreateOrGet(context.Background(), "https://occupied.example", "aaaaaaaaaa")
	if err != nil {
		t.Fatalf("seed storage error = %v", err)
	}
	generator := &sequenceGenerator{codes: []string{"aaaaaaaaaa", "bbbbbbbbbb"}}
	svc, err := New(store, generator, "http://localhost:8080", 2)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	result, err := svc.Create(context.Background(), "https://new.example")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Link.Code != "bbbbbbbbbb" {
		t.Fatalf("Create() code = %q, want bbbbbbbbbb", result.Link.Code)
	}
}

func TestCreateRejectsInvalidURL(t *testing.T) {
	svc, err := New(memory.New(), &sequenceGenerator{codes: []string{"aaaaaaaaaa"}}, "http://localhost:8080", 10)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = svc.Create(context.Background(), "javascript:alert(1)")
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("Create() error = %v, want ErrInvalidURL", err)
	}
}

func TestGetRejectsInvalidCode(t *testing.T) {
	svc, err := New(memory.New(), &sequenceGenerator{codes: []string{"aaaaaaaaaa"}}, "http://localhost:8080", 10)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = svc.Get(context.Background(), "short")
	if !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("Get() error = %v, want ErrInvalidCode", err)
	}
}

func TestCreateReturnsGeneratorError(t *testing.T) {
	expected := errors.New("generator failed")
	svc, err := New(memory.New(), &sequenceGenerator{err: expected}, "http://localhost:8080", 10)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = svc.Create(context.Background(), "https://example.com")
	if !errors.Is(err, expected) {
		t.Fatalf("Create() error = %v, want wrapped generator error", err)
	}
}

func TestCreateReturnsGenerationExhausted(t *testing.T) {
	store := memory.New()
	_, _, err := store.CreateOrGet(context.Background(), "https://occupied.example", "aaaaaaaaaa")
	if err != nil {
		t.Fatalf("seed storage error = %v", err)
	}
	svc, err := New(store, &sequenceGenerator{codes: []string{"aaaaaaaaaa"}}, "http://localhost:8080", 2)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = svc.Create(context.Background(), "https://new.example")
	if !errors.Is(err, ErrGenerationExhausted) {
		t.Fatalf("Create() error = %v, want ErrGenerationExhausted", err)
	}
}
