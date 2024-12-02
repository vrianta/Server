package server

import (
	"fmt"
	"net/http"
	"time"
)

func GetCookie(cookie_name string, r *http.Request) *http.Cookie {
	if cookie, err := r.Cookie(cookie_name); err == nil {
		// Log("Cookie: ", cookie)
		return cookie
	}

	return nil
}

// Return true if it created te cookie else false means cookie is already present in the session
func AddCookie(cookie_config *http.Cookie, w http.ResponseWriter, r *http.Request) {
	cookie_header := formHeader(cookie_config)
	WriteConsole("cookie header: ", cookie_header)
	w.Header().Add("Set-Cookie", cookie_header)
	w.Header().Add("X-Custom-Header", "MyHeaderValue")
	w.Header().Set("Set-Cookie", cookie_header)
}

func RemoveCookie(cookie_name string, w http.ResponseWriter, r *http.Request) {
	cookie_header := fmt.Sprintf("%s=expire; Max-Age=-1; Expires=%s;", cookie_name, time.Now().UTC().Format(http.TimeFormat))
	if _, err := r.Cookie(cookie_name); err == nil {
		w.Header().Add("Set-Cookie", cookie_header)
	}
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
