package auth

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

const (
	// SessionTokenLength is the number of random bytes for session tokens.
	SessionTokenLength = 32
	// BcryptCost is the cost factor for bcrypt hashing.
	BcryptCost = 12
)

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a password with a bcrypt hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSessionToken creates a cryptographically secure random session token.
func GenerateSessionToken() (string, error) {
	b := make([]byte, SessionTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateRandomPassword creates a cryptographically secure random password.
func GenerateRandomPassword() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:20], nil
}
