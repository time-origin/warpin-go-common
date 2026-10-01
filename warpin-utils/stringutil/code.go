package stringutil

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	// EntityCodeLength is the fixed length of generated business entity codes.
	EntityCodeLength   = 12
	entityCodeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// NormalizeOrGenerateCode returns an uppercase, trimmed caller-supplied code,
// or a cryptographically random 12-character alphanumeric code when absent.
func NormalizeOrGenerateCode(value string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized != "" {
		return normalized, nil
	}

	code := make([]byte, EntityCodeLength)
	alphabetLength := big.NewInt(int64(len(entityCodeAlphabet)))
	for i := range code {
		index, err := rand.Int(rand.Reader, alphabetLength)
		if err != nil {
			return "", fmt.Errorf("generate entity code: %w", err)
		}
		code[i] = entityCodeAlphabet[index.Int64()]
	}
	return string(code), nil
}
