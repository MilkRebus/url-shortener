package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/milkrebus/url-shortener/internal/service"
	"github.com/milkrebus/url-shortener/internal/storage"
)

type fakeService struct {
	create func(context.Context, string) (service.CreateResult, error)
	get    func(context.Context, string) (storage.Link, error)
}

func (f fakeService) Create(ctx context.Context, originalURL string) (service.CreateResult, error) {
	return f.create(ctx, originalURL)
}

func (f fakeService) Get(ctx context.Context, code string) (storage.Link, error) {
	return f.get(ctx, code)
}

func TestCreateLink(t *testing.T) {
	handler := New(fakeService{
		create: func(_ context.Context, originalURL string) (service.CreateResult, error) {
			if originalURL != "https://example.com" {
				t.Fatalf("Create() URL = %q", originalURL)
			}
			return service.CreateResult{
				Link:     storage.Link{Code: "aB3_q9ZxK2", OriginalURL: originalURL, CreatedAt: time.Now()},
				ShortURL: "http://localhost:8080/aB3_q9ZxK2",
				Created:  true,
			}, nil
		},
		get: func(context.Context, string) (storage.Link, error) {
			return storage.Link{}, storage.ErrNotFound
		},
	}, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/links", strings.NewReader(`{"url":"https://example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"aB3_q9ZxK2"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestCreateLinkRejectsInvalidJSON(t *testing.T) {
	handler := New(fakeService{
		create: func(context.Context, string) (service.CreateResult, error) {
			t.Fatal("Create() must not be called")
			return service.CreateResult{}, nil
		},
		get: func(context.Context, string) (storage.Link, error) {
			return storage.Link{}, nil
		},
	}, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/links", strings.NewReader(`{"url":`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestGetLink(t *testing.T) {
	handler := New(fakeService{
		create: func(context.Context, string) (service.CreateResult, error) {
			return service.CreateResult{}, nil
		},
		get: func(_ context.Context, code string) (storage.Link, error) {
			if code != "aB3_q9ZxK2" {
				t.Fatalf("Get() code = %q", code)
			}
			return storage.Link{Code: code, OriginalURL: "https://example.com"}, nil
		},
	}, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/links/aB3_q9ZxK2", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"url":"https://example.com"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestGetLinkNotFound(t *testing.T) {
	handler := New(fakeService{
		create: func(context.Context, string) (service.CreateResult, error) {
			return service.CreateResult{}, nil
		},
		get: func(context.Context, string) (storage.Link, error) {
			return storage.Link{}, storage.ErrNotFound
		},
	}, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/links/aB3_q9ZxK2", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestRedirect(t *testing.T) {
	handler := New(fakeService{
		create: func(context.Context, string) (service.CreateResult, error) {
			return service.CreateResult{}, nil
		},
		get: func(context.Context, string) (storage.Link, error) {
			return storage.Link{Code: "aB3_q9ZxK2", OriginalURL: "https://example.com/path"}, nil
		},
	}, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/aB3_q9ZxK2", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusFound)
	}
	if location := response.Header().Get("Location"); location != "https://example.com/path" {
		t.Fatalf("Location = %q", location)
	}
}

func TestReadinessFailure(t *testing.T) {
	handler := New(fakeService{
		create: func(context.Context, string) (service.CreateResult, error) {
			return service.CreateResult{}, nil
		},
		get: func(context.Context, string) (storage.Link, error) {
			return storage.Link{}, nil
		},
	}, func(context.Context) error {
		return errors.New("database unavailable")
	}, nil)
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
