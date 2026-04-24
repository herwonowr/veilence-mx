// Package hasher provides password hashing implementations.
package hasher

import (
	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher implements password hashing using bcrypt.
type BcryptHasher struct{}

// New creates a new BcryptHasher.
func New() *BcryptHasher {
	return &BcryptHasher{}
}

// Hash produces a bcrypt hash of the given password.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Compare checks whether the given password matches the stored bcrypt hash.
// Returns nil on match, error otherwise.
func (h *BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
