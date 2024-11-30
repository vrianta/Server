package Server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

var (
	cookieName = "sessionid"
)

// return true server created a new session
func StartSession(userid string) string {

	// Check if the session in their with the user if the session present this will do
	if sessionID := GetSessionID(); sessionID != "" {
		Log("Session Already present in the user lavel")
		if session, ok := sessionStore[sessionID]; !ok {
			Log("Session present in the user section but not in server hence removing the session and going to ask user to re-login")
			EndSession()
			http.Redirect(W, R, "/", http.StatusFound)
			return sessionID
		} else if uid, uid_ok := session["userID"]; uid_ok && uid != userid && userid != "Guest" {
			EndSession()
			if newSessionID, err := GenerateSessionID(); err == nil {
				sessionStore[newSessionID] = map[string]interface{}{
					"userID":    userid,
					"sessionID": newSessionID,
				}
				SESSION = sessionStore[newSessionID]
				return sessionID
				// Storing in the Session
			}
			return sessionID
		} else {
			Log("Session is perfectly fine returning false")
			return sessionID
		}
	}

	if sessionID, err := GenerateSessionID(); err == nil {
		// storeMutex.Lock()
		sessionStore[sessionID] = map[string]interface{}{
			"userID":    userid,
			"sessionID": sessionID,
		}
		SESSION = sessionStore[sessionID]
		// storeMutex.Unlock()
		Log(SESSION)

		if userid == "Guest" {
			AddCookie(&http.Cookie{
				Name:    cookieName,
				Value:   sessionID,
				Expires: time.Now().Add(5 * time.Minute).UTC(),
			})
		} else {
			Log("Creating Cookie for sessionID")
			AddCookie(&http.Cookie{
				Name:  cookieName,
				Value: sessionID,
				// Expires: time.Now().Add(5 * time.Minute).UTC(),
			})
		}
		return sessionID
	}

	return ""
}

func EndSession() {

	if sessionID := GetSessionID(); sessionID == "" {
		println("No Session to be END... Hense Creating new session")
		return
	} else {
		// Remove session data from the store
		// storeMutex.Lock()
		delete(sessionStore, sessionID)
		// storeMutex.Unlock()

		RemoveCookie(cookieName)
	}
}

func GetSessionID() string {
	if cookie := GetCookie(cookieName); cookie != nil { // means cookie is already present int he system
		Log("Cookie Value", cookie.Value)
		return cookie.Value
	} else {
		return ""
	}
}

func IsLoggedIn() bool {

	if sessionID := GetSessionID(); sessionID == "" {
		Log("No User Session is available to look for")
		return false
	} else if value, ok := SESSION["isLoggedIn"]; ok {
		Log("User is Logged in")
		return value.(bool)
	} else {
		Log("No User Logged in")
		return false
	}
}

// GenerateRandomToken generates a random token of specified length
func GenerateRandomToken(length int) (string, error) {
	// Create a byte slice with the desired length
	bytes := make([]byte, length)
	// Fill the slice with random bytes
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	// Convert the random bytes to a hex string
	return hex.EncodeToString(bytes), nil
}

func GenerateSessionID() (string, error) {
	// Create a byte slice to hold the random data
	bytes := make([]byte, 16) // 16 bytes = 128 bits, which is a reasonable length for a session ID

	// Generate random bytes using crypto/rand
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err // Return an error if random generation fails
	}

	// Convert the byte slice to a hexadecimal string
	sessionID := hex.EncodeToString(bytes)

	return sessionID, nil
}
