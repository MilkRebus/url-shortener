package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/milkrebus/url-shortener/internal/service"
	"github.com/milkrebus/url-shortener/internal/storage"
)

const maxRequestBody = 1 << 20

type LinkService interface {
	Create(ctx context.Context, originalURL string) (service.CreateResult, error)
	Get(ctx context.Context, code string) (storage.Link, error)
}

type Handler struct {
	service   LinkService
	readiness func(context.Context) error
	logger    *slog.Logger
}

func New(linkService LinkService, readiness func(context.Context) error, logger *slog.Logger) http.Handler {
	if readiness == nil {
		readiness = func(context.Context) error { return nil }
	}
	if logger == nil {
		logger = slog.Default()
	}
	handler := &Handler{service: linkService, readiness: readiness, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/links", handler.createLink)
	mux.HandleFunc("GET /api/v1/links/{code}", handler.getLink)
	mux.HandleFunc("GET /health/live", handler.liveness)
	mux.HandleFunc("GET /health/ready", handler.readinessCheck)
	mux.HandleFunc("GET /{code}", handler.redirect)
	return handler.recoverPanic(handler.securityHeaders(mux))
}

type createLinkRequest struct {
	URL string `json:"url"`
}

type createLinkResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

type getLinkResponse struct {
	URL string `json:"url"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) createLink(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request createLinkRequest
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "request body must contain one JSON object"})
		return
	}
	result, err := h.service.Create(r.Context(), request.URL)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	writeJSON(w, status, createLinkResponse{Code: result.Link.Code, ShortURL: result.ShortURL})
}

func (h *Handler) getLink(w http.ResponseWriter, r *http.Request) {
	link, err := h.service.Get(r.Context(), r.PathValue("code"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, getLinkResponse{URL: link.OriginalURL})
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request) {
	link, err := h.service.Get(r.Context(), r.PathValue("code"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	http.Redirect(w, r, link.OriginalURL, http.StatusFound)
}

func (h *Handler) liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) readinessCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if err := h.readiness(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "service is not ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidURL), errors.Is(err, service.ErrInvalidCode):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, storage.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: storage.ErrNotFound.Error()})
	case errors.Is(err, service.ErrGenerationExhausted):
		h.logger.Error("unique code generation exhausted", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "could not create short link"})
	default:
		h.logger.Error("request failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func (h *Handler) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.Error("panic recovered", "value", recovered)
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("extra JSON data")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
