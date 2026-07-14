package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/milkrebus/url-shortener/internal/storage"

	"github.com/jackc/pgx/v5"
)

type fakeDatabase struct {
	rows  []pgx.Row
	index int
}

func (f *fakeDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	row := f.rows[f.index]
	f.index++
	return row
}

func (*fakeDatabase) Ping(context.Context) error { return nil }
func (*fakeDatabase) Close()                     {}

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, value := range r.values {
		switch target := dest[i].(type) {
		case *string:
			*target = value.(string)
		case *time.Time:
			*target = value.(time.Time)
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}

func TestCreateOrGetCreatesLink(t *testing.T) {
	createdAt := time.Now().UTC()
	store := &Storage{pool: &fakeDatabase{rows: []pgx.Row{
		fakeRow{values: []any{"aB3_q9ZxK2", "https://example.com", createdAt}},
	}}}
	link, created, err := store.CreateOrGet(context.Background(), "https://example.com", "aB3_q9ZxK2")
	if err != nil {
		t.Fatalf("CreateOrGet() error = %v", err)
	}
	if !created {
		t.Fatal("CreateOrGet() created = false, want true")
	}
	if link.Code != "aB3_q9ZxK2" {
		t.Fatalf("CreateOrGet() code = %q", link.Code)
	}
}

func TestCreateOrGetReturnsExistingURL(t *testing.T) {
	createdAt := time.Now().UTC()
	store := &Storage{pool: &fakeDatabase{rows: []pgx.Row{
		fakeRow{err: pgx.ErrNoRows},
		fakeRow{values: []any{"existing01", "https://example.com", createdAt}},
	}}}
	link, created, err := store.CreateOrGet(context.Background(), "https://example.com", "proposed01")
	if err != nil {
		t.Fatalf("CreateOrGet() error = %v", err)
	}
	if created {
		t.Fatal("CreateOrGet() created = true, want false")
	}
	if link.Code != "existing01" {
		t.Fatalf("CreateOrGet() code = %q, want existing01", link.Code)
	}
}

func TestCreateOrGetReportsCodeCollision(t *testing.T) {
	store := &Storage{pool: &fakeDatabase{rows: []pgx.Row{
		fakeRow{err: pgx.ErrNoRows},
		fakeRow{err: pgx.ErrNoRows},
	}}}
	_, _, err := store.CreateOrGet(context.Background(), "https://example.com", "aB3_q9ZxK2")
	if !errors.Is(err, storage.ErrCodeCollision) {
		t.Fatalf("CreateOrGet() error = %v, want ErrCodeCollision", err)
	}
}

func TestGetByCodeNotFound(t *testing.T) {
	store := &Storage{pool: &fakeDatabase{rows: []pgx.Row{
		fakeRow{err: pgx.ErrNoRows},
	}}}
	_, err := store.GetByCode(context.Background(), "aB3_q9ZxK2")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetByCode() error = %v, want ErrNotFound", err)
	}
}
