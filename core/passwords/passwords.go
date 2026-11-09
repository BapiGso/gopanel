package passwords

import (
	"crypto/subtle"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash for a plaintext credential.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword reports whether supplied matches stored. Stored values that
// are not bcrypt hashes are treated as legacy plaintext credentials.
func VerifyPassword(stored, supplied string) bool {
	if !strings.HasPrefix(stored, "$2a$") &&
		!strings.HasPrefix(stored, "$2b$") &&
		!strings.HasPrefix(stored, "$2y$") {
		return subtle.ConstantTimeCompare([]byte(stored), []byte(supplied)) == 1
	}
	return bcrypt.CompareHashAndPassword([]byte(stored), []byte(supplied)) == nil
}
