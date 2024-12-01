package Server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

var (
	cookieName = "sessionid"
)

// StartSession attempts to retrieve or create a new session
func StartSession() string {
	// Check if a session exists for the user
	if sessionID := GetSessionID(); sessionID != "" {
		Log("Session already present at the user level")

		// Validate the session exists in the store
		if session, ok := sessionStore[sessionID]; ok && len(session) != 0 {
			SESSION = sessionStore[sessionID]
			Log("Session is valid, returning sessionID")
			return sessionID
		}
	}

	// No session exists, create a new one
	return createNewSession()
}

// Creates a new session and sets cookies
func createNewSession() string {
	// Generate a session ID
	sessionID, err := GenerateSessionID()
	if err != nil {
		Log("Error generating session ID:", err)
		return ""
	}

	// Lock the store and add the new session
	storeMutex.Lock()
	sessionStore[sessionID] = make(map[string]interface{})
	SESSION = sessionStore[sessionID]
	storeMutex.Unlock()

	Log("New session created:", sessionID)
	setSessionCookie(sessionID)
	return sessionID
}

// Sets the session cookie in the client's browser
func setSessionCookie(sessionID string) {
	// Set cookie with expiration time for 30 minutes
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    sessionID,
		HttpOnly: true,
		// Expiry time can be set here if necessary
		// Expires: time.Now().Add(30 * time.Minute).UTC(),
	}

	AddCookie(cookie)
	Log("Session cookie set with session ID:", sessionID)
}

// EndSession ends the current user session
func EndSession() {
	sessionID := GetSessionID()
	if sessionID == "" {
		Log("No active session found, cannot end session.")
		return
	}

	// Remove session data from the store
	Log("Ending session for session ID:", sessionID)
	storeMutex.Lock()
	delete(sessionStore, sessionID)
	storeMutex.Unlock()

	RemoveCookie(cookieName)
	Log("Session ended and cookie removed:", sessionID)
}

// GetSessionID retrieves the session ID from the client's cookie
func GetSessionID() string {
	cookie := GetCookie(cookieName)
	if cookie != nil {
		Log("Session cookie found with value:", cookie.Value)
		return cookie.Value
	}
	Log("No session cookie found.")
	return ""
}

// IsLoggedIn checks if the user is logged in by verifying the session
func IsLoggedIn() bool {
	sessionID := GetSessionID()
	if sessionID == "" {
		Log("No session found, user is not logged in.")
		return false
	}

	if value, ok := SESSION["isLoggedIn"]; ok && value.(bool) {
		Log("User is logged in with session ID:", sessionID)
		return true
	}

	Log("User is not logged in, session ID:", sessionID)
	return false
}

// GenerateRandomToken generates a random token for the user
func GenerateRandomToken(userID string) (string, error) {
	// It's better to load the secret key from a secure place rather than hardcoding it
	secretKey := "yuhjlthushxsiookj98sans"
	length := 64

	// Generate random bytes
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		Log("Error generating random bytes:", err)
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
	Log("Generated token for user ID:", userID)
	return token, nil
}

// GenerateSessionID generates a random session ID
func GenerateSessionID() (string, error) {
	// Create a byte slice to hold the random data
	bytes := make([]byte, 16) // 16 bytes = 128 bits, reasonable for session ID

	// Generate random bytes using crypto/rand
	_, err := rand.Read(bytes)
	if err != nil {
		Log("Error generating random session ID:", err)
		return "", err
	}

	// Convert the byte slice to a hexadecimal string
	sessionID := hex.EncodeToString(bytes)
	Log("Generated session ID:", sessionID)
	return sessionID, nil
}
