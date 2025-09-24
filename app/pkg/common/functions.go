package common

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)

// RemoveValue removes an element from a slice by its value.
func RemoveValueSlice(slice []string, value string) []string {
	for i, v := range slice {
		if v == value {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// Hash a password
func HashPassword(password string) string {
	salt := make([]byte, 16) // Generate a random salt
	if _, err := rand.Read(salt); err != nil {
		return ""
	}
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	// Concatenate salt and hash
	saltedHash := make([]byte, 16+len(hash))
	copy(saltedHash, salt)
	copy(saltedHash[16:], hash)
	return base64.StdEncoding.EncodeToString(saltedHash)
}

func CheckPassword(passwordLogin string, passwordDB string) bool {
	passwordDBHash, err := base64.StdEncoding.DecodeString(passwordDB)
	if err != nil {
		return false
	}
	// Extract salt and stored hash
	salt := passwordDBHash[:16]
	storedHash := passwordDBHash[16:]
	passwordLoginHash := argon2.IDKey([]byte(passwordLogin), salt, 3, 64*1024, 4, 32)

	return subtle.ConstantTimeCompare(passwordLoginHash, storedHash) == 1
}
