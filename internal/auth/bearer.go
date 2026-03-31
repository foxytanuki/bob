package auth

import (
	"crypto/subtle"
	"strings"
)

func ValidateBearerToken(header, expected string) bool {
	if expected == "" {
		return false
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}

	provided := strings.TrimPrefix(header, prefix)
	if len(provided) != len(expected) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
