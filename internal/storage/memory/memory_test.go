package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/milkrebus/url-shortener/internal/storage"
)

func TestCreateOrGetAndGetByCode(t *testing.T) {
	store := New()
	ctx := context.Background()
	createdLink, created, err := store.CreateOrGet(ctx, "https://example.com", "aB3_q9ZxK2")
	if err != nil {
		t.Fatalf("CreateOrGet() error = %v", err)
	}
	if !created {
		t.Fatal("CreateOrGet() created = false, want true")
	}
	got, err := store.GetByCode(ctx, createdLink.Code)
	if err != nil {
		t.Fatalf("GetByCode() error = %v", err)
	}
	if got.OriginalURL != createdLink.OriginalURL {
		t.Fatalf("GetByCode() URL = %q, want %q", got.OriginalURL, createdLink.OriginalURL)
	}
}

func TestSameURLReturnsExistingCode(t *testing.T) {
	store := New()
	ctx := context.Background()
	first, _, err := store.CreateOrGet(ctx, "https://example.com", "aaaaaaaaaa")
	if err != nil {
		t.Fatalf("first CreateOrGet() error = %v", err)
	}
	second, created, err := store.CreateOrGet(ctx, "https://example.com", "bbbbbbbbbb")
	if err != nil {
		t.Fatalf("second CreateOrGet() error = %v", err)
	}
	if created {
		t.Fatal("second CreateOrGet() created = true, want false")
	}
	if second.Code != first.Code {
		t.Fatalf("second code = %q, want %q", second.Code, first.Code)
	}
}

func TestCodeCollision(t *testing.T) {
	store := New()
	ctx := context.Background()
	_, _, err := store.CreateOrGet(ctx, "https://first.example", "aaaaaaaaaa")
	if err != nil {
		t.Fatalf("first CreateOrGet() error = %v", err)
	}
	_, _, err = store.CreateOrGet(ctx, "https://second.example", "aaaaaaaaaa")
	if !errors.Is(err, storage.ErrCodeCollision) {
		t.Fatalf("second CreateOrGet() error = %v, want ErrCodeCollision", err)
	}
}

func TestConcurrentSameURLCreatesOneLink(t *testing.T) {
	store := New()
	ctx := context.Background()
	const workers = 100
	results := make(chan string, workers)
	errorsCh := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			code := fmt.Sprintf("code%06d", index)
			link, _, err := store.CreateOrGet(ctx, "https://example.com/shared", code)
			if err != nil {
				errorsCh <- err
				return
			}
			results <- link.Code
		}(i)
	}
	wg.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		t.Fatalf("concurrent CreateOrGet() error = %v", err)
	}
	var expected string
	for code := range results {
		if expected == "" {
			expected = code
		}
		if code != expected {
			t.Fatalf("got multiple codes: %q and %q", expected, code)
		}
	}
}

func TestGetByCodeNotFound(t *testing.T) {
	_, err := New().GetByCode(context.Background(), "aaaaaaaaaa")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetByCode() error = %v, want ErrNotFound", err)
	}
}
