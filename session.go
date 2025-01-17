package server

import (
	"fmt"
	"net/http"
	"time"
)

type SESSIONTYPE map[string]interface{}
type POSTTYPE map[string]string
type GETTYPE map[string]string
type SessionHandler struct {
	ID string
	W  http.ResponseWriter
	R  *http.Request

	POST POSTTYPE
	GET  GETTYPE
	VAR  SESSIONTYPE

	Renderhandler RenderHandeler
}

func NewSessionHandlerOBJ(w http.ResponseWriter, r *http.Request) *SessionHandler {
	return &SessionHandler{
		W:    w,
		R:    r,
		POST: make(POSTTYPE),
		GET:  make(GETTYPE),
		VAR: SESSIONTYPE{
			"uid":        "Guest",
			"isLoggedIn": false,
		},

		Renderhandler: NewRenderHandlerObj(w),
	}
}

func GetSessionID(r *http.Request) *string {
	cookie := GetCookie("sessionid", r)
	if cookie != nil {
		fmt.Println("Session cookie found with value:", cookie.Value)
		return &cookie.Value
	}
	return nil
}

func (sh *SessionHandler) Login(uid string) {
	WriteConsole("Attempting to Login")
	sh.VAR["uid"] = uid
	sh.VAR["isLoggedIn"] = true
	// If no valid session ID is found, create a new session
	sh.SetSessionCookie(&sh.ID)
}

func (sh *SessionHandler) IsLoggedIn() bool {
	if isloggedIn, ok := sh.VAR["isLoggedIn"]; ok {
		return isloggedIn.(bool)
	}
	return false
}

// StartSession attempts to retrieve or create a new session
func (sh *SessionHandler) StartSession() *string {
	WriteConsole("Attempting to start a session")

	// Try to get an existing session ID from the request
	if sessionID := GetSessionID(sh.R); sessionID != nil {
		WriteConsole("Session ID found in request: ", *sessionID)
		if *sessionID == "expire" {
			return sh.CreateNewSession()
		}
		// If the session ID doesn't match the current handler's ID, create a new session
		if (*sessionID) != sh.ID {
			WriteConsole("Session ID from request does not match handler's session ID. Creating a new session.", *sessionID, " : ", sh.ID)
			EndSession(sh.W, *sh.R, sh)
			return nil
		} else {
			WriteConsole("Session ID from request matches the handler's session ID. Using the existing session.")
		}
	} else {
		WriteConsole("No session ID found in request. Creating a new session.")
	}

	// If no valid session ID is found, create a new session
	return sh.CreateNewSession()
}

func (sh *SessionHandler) UpdateSession(_w *http.ResponseWriter, _r *http.Request) {
	sh.W = *_w
	sh.R = _r

	sh.Renderhandler.W = *_w
}

// Creates a new session and sets cookies
func (sh *SessionHandler) CreateNewSession() *string {
	// Generate a session ID
	sessionID, err := GenerateSessionID()
	if err != nil {
		return nil
	}

	sh.ID = sessionID
	sh.SetSessionCookie(&sessionID)

	return &sessionID
}

// Sets the session cookie in the client's browser
func (sh *SessionHandler) SetSessionCookie(sessionID *string) {
	c := &http.Cookie{
		Name:     "sessionid",
		Value:    *sessionID,
		HttpOnly: true,
		Expires:  time.Now().Add(30 * time.Minute).UTC(),
	}
	AddCookie(c, sh.W, sh.R)
}

func EndSession(w http.ResponseWriter, r http.Request, sessionhandler *SessionHandler) {
	sessionID := GetSessionID(&r)

	if sessionID == nil {
		WriteConsole("No active session found, cannot end session.")
		return
	}

	// Remove session data from the store
	WriteConsole("Ending session for session ID:", *sessionID)

	RemoveCookie("sessionid", w, &r)
	RemoveSessionHandler(sessionID)
}

func (sh *SessionHandler) RequestHandler() {
	// Initialize queryParams once for later use
	queryParams := sh.R.URL.Query()

	sh.POST = make(POSTTYPE)
	sh.GET = make(GETTYPE)

	// Check if the request method is POST
	if sh.R.Method == http.MethodPost {
		// Parse multipart form data with a 10 MB limit for file uploads
		err := sh.R.ParseMultipartForm(10 << 20) // 10 MB
		if err != nil {
			WriteConsole("Error parsing multipart form data: ", err)
			// http.Error(sh.W, "Error parsing multipart form data", http.StatusBadRequest)
		}
		WriteConsole("Handling POST request")
		// Handle POST form data
		for key, values := range sh.R.PostForm {
			sh.HandlePostParams(key, values)
		}
	}

	// Log handling of query parameters for non-POST methods
	WriteConsole("Handling non-POST request, processing query parameters")
	for key, values := range queryParams {
		sh.HandleQueryParams(key, values)
	}
}

// handleQueryParams processes parameters found in the URL query
func (sh *SessionHandler) HandleQueryParams(key string, values []string) {
	var err error
	// Check for multiple values

	if len(values) > 1 {
		if sh.GET[key], err = StringArrayToJson(values); err != nil {
			// WriteConsole("Failed to convert multiple values of key '", key, "' to JSON: ", key, err)
			http.Error(sh.W, "Failed to convert data to JSON", http.StatusMethodNotAllowed)

		}
	} else {
		sh.GET[key] = values[0] // Store single value as a string
	}
	// WriteConsole("Handled query parameter - key: ", key, ", value: ", sh.GET[key])
}

// handlePostParams processes parameters found in the POST data
func (sh *SessionHandler) HandlePostParams(key string, values []string) {
	var err error
	// Check for multiple values
	if len(values) > 1 {
		if sh.POST[key], err = StringArrayToJson(values); err != nil {
			// WriteConsole("Failed to convert multiple values of key '", key, "' to JSON: ", err)
			http.Error(sh.W, "Failed to convert data to JSON", http.StatusMethodNotAllowed)
		}
	} else {
		sh.POST[key] = values[0] // Store single value as a string
	}
	// WriteConsole("Handled POST parameter - key: ", key, ", value: ", sh.POST[key])
}
