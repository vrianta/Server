package Server

import (
	"fmt"
	"net/http"
	"time"
)

// Return true if it created te cookie else false means cookie is already present in the session
func AddCookie(cookie_config *http.Cookie) bool {

	// Add the Set-Cookie header
	if _, err := R.Cookie(cookie_config.Name); err != nil {
		cookie_header := formHeader(cookie_config)
		Log("Cookie Header: ", cookie_header)
		W.Header().Add("Set-Cookie", cookie_header)
		W.Header().Add("X-Custom-Header", "MyHeaderValue")
		W.Header().Set("Set-Cookie", cookie_header)
		return true
	}

	return false
}

func RemoveCookie(cookie_name string) {
	cookie_header := fmt.Sprintf("%s=expire; Max-Age=-1; Expires=%s;", cookie_name, time.Now().UTC().Format(http.TimeFormat))
	Log("Cookie header:", cookie_header)
	if _, err := R.Cookie(cookie_name); err == nil {
		W.Header().Add("Set-Cookie", cookie_header)
	}
}

func GetCookie(cookie_name string) *http.Cookie {
	if cookie, err := R.Cookie(cookie_name); err == nil {
		// Log("Cookie: ", cookie)
		return cookie
	}

	return nil
}

func formHeader(cookie_config *http.Cookie) string {

	cookie_header := fmt.Sprintf(
		"%s=%s;",
		cookie_config.Name,
		cookie_config.Value,
	)

	if !cookie_config.Expires.IsZero() {
		cookie_header += fmt.Sprintf(" Expires=%s;", cookie_config.Expires.Format(http.TimeFormat))
	} else if cookie_config.MaxAge == -1 {
		cookie_header += fmt.Sprintf(" Max-Age=%d;", 0)
	}
	if cookie_config.HttpOnly {
		cookie_header += " HttpOnly;"
	}
	if cookie_config.Secure {
		cookie_header += " Secure;"
	}

	return cookie_header
}
