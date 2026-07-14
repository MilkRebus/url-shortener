package config

import "testing"

func TestLoadMemoryConfig(t *testing.T) {
	cfg, err := Load([]string{"-storage=memory", "-base-url=http://localhost:8080"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.StorageType != "memory" {
		t.Fatalf("StorageType = %q, want memory", cfg.StorageType)
	}
}

func TestLoadRejectsUnknownStorage(t *testing.T) {
	_, err := Load([]string{"-storage=redis"})
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := Load([]string{"-storage=postgres", "-database-url="})
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadRejectsInvalidEnvironmentValue(t *testing.T) {
	t.Setenv("DB_MAX_CONNS", "not-a-number")
	_, err := Load(nil)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
