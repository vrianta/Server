package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"golang.org/x/crypto/bcrypt"
)

// GenerateRandomToken generates a random token for the user
func GenerateRandomToken(userID string) (string, error) {
	// It's better to load the secret key from a secure place rather than hardcoding it
	secretKey := "yuhjlthushxsiookj98sans"
	length := 64

	// Generate random bytes
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Create an HMAC using the secret key and random bytes
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write(randomBytes)
	h.Write([]byte(userID))

	// Compute the final HMAC value
	finalHash := h.Sum(nil)

	// Return the hex-encoded hash as the token
	token := hex.EncodeToString(finalHash)
	return token, nil
}

// GenerateSessionID generates a random session ID
func GenerateSessionID() (string, error) {
	// Create a byte slice to hold the random data
	bytes := make([]byte, 16) // 16 bytes = 128 bits, reasonable for session ID

	// Generate random bytes using crypto/rand
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Convert the byte slice to a hexadecimal string
	sessionID := hex.EncodeToString(bytes)
	return sessionID, nil
}

// stringToJson converts a slice of strings to a JSON-encoded string
func StringArrayToJson(data []string) (string, error) {
	// Marshal the slice of strings into a JSON-encoded byte slice
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Return the JSON string (as a string)
	return string(jsonData), nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password *string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a hashed password with a plain password
func CheckPassword(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil // Returns true if passwords match
}
