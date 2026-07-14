package generator

import (
	"crypto/rand"
	"fmt"
	"io"
)

const (
	Alphabet   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	CodeLength = 10
)

type Generator interface {
	Generate() (string, error)
}

type Random struct {
	reader io.Reader
}

func NewRandom() *Random {
	return &Random{reader: rand.Reader}
}

func NewRandomWithReader(reader io.Reader) *Random {
	return &Random{reader: reader}
}

func (g *Random) Generate() (string, error) {
	if g.reader == nil {
		return "", fmt.Errorf("random source is nil")
	}
	result := make([]byte, 0, CodeLength)
	buffer := make([]byte, 16)
	limit := byte(252)
	for len(result) < CodeLength {
		if _, err := io.ReadFull(g.reader, buffer); err != nil {
			return "", fmt.Errorf("read random bytes: %w", err)
		}
		for _, value := range buffer {
			if value >= limit {
				continue
			}
			result = append(result, Alphabet[int(value)%len(Alphabet)])
			if len(result) == CodeLength {
				break
			}
		}
	}
	return string(result), nil
}

func IsValidCode(code string) bool {
	if len(code) != CodeLength {
		return false
	}
	for i := 0; i < len(code); i++ {
		value := code[i]
		if (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9') || value == '_' {
			continue
		}
		return false
	}
	return true
}
