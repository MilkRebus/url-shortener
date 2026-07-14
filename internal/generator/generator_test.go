package generator

import (
	"errors"
	"strings"
	"testing"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("random source failed")
}

func TestGenerateProducesValidCode(t *testing.T) {
	reader := strings.NewReader("0123456789abcdef")
	code, err := NewRandomWithReader(reader).Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !IsValidCode(code) {
		t.Fatalf("Generate() returned invalid code %q", code)
	}
}

func TestGenerateReturnsReaderError(t *testing.T) {
	_, err := NewRandomWithReader(failingReader{}).Generate()
	if err == nil {
		t.Fatal("Generate() error = nil, want error")
	}
}

func TestIsValidCode(t *testing.T) {
	tests := []struct {
		code  string
		valid bool
	}{
		{code: "aB3_q9ZxK2", valid: true},
		{code: "short", valid: false},
		{code: "abcdefghij!", valid: false},
		{code: "abcdefgh-1", valid: false},
	}
	for _, test := range tests {
		if got := IsValidCode(test.code); got != test.valid {
			t.Errorf("IsValidCode(%q) = %v, want %v", test.code, got, test.valid)
		}
	}
}
