package server

import (
	"net/http"
)

// var storeMutex

func (sh *ServerHandlerType) routingHandler(w http.ResponseWriter, r *http.Request) {

	// Log the incoming request URL
	WriteConsole("Received request for URL: ", r.URL.Path)

	sessionID := GetSessionID(r)
	if sessionID == nil { // means no session has been established with the user
		WriteConsole("No session found, starting a new session")

		sessionHandler := NewSessionHandlerOBJ(w, r)
		sessionID = sessionHandler.StartSession()
		if sessionID != nil {
			WriteConsolef("New session started with ID: %s \n", *sessionID)
			sh.SessionHandler[(*sessionID)] = *sessionHandler
			if value, ok := sh.Routes[r.URL.Path]; ok {
				WriteConsolef("Route found for URL: %s, calling handler \n", r.URL.Path)
				sessionHandler.UpdateSession(&w, r)
				sessionHandler.RequestHandler()
				value(sessionHandler)
				sessionHandler.Renderhandler.StartRender()
			} else {
				WriteConsolef("Route not found for URL: %s \n", r.URL.Path)
				http.Error(w, "404 Error : Route not found ", 404)
			}
		} else {
			WriteConsole("Failed to start session")
			return
		}
	} else {
		WriteConsolef("Session ID found: %s \n", *sessionID)

		if sessionHandler, ok := sh.SessionHandler[(*sessionID)]; ok { // session is already created
			WriteConsole("Session exists, processing request")

			if value, ok := sh.Routes[r.URL.Path]; ok {
				WriteConsolef("Route found for URL: %s, calling handler\n", r.URL.Path)
				sessionHandler.UpdateSession(&w, r)
				sessionHandler.RequestHandler()
				value(&sessionHandler)
				sessionHandler.Renderhandler.StartRender()
			} else {
				WriteConsolef("Route not found for URL: %s\n", r.URL.Path)
				http.Error(w, "404 Error : Route not found ", 404)
			}

		} else {
			WriteConsole("Session does not exist in SessionHandler, creating a new one")

			sessionHandler := NewSessionHandlerOBJ(w, r)
			sessionID = sessionHandler.StartSession()
			if sessionID != nil {
				WriteConsolef("New session started with ID: %s\n", *sessionID)
				sh.SessionHandler[(*sessionID)] = *sessionHandler

				if value, ok := sh.Routes[r.URL.Path]; ok {
					WriteConsolef("Route found for URL: %s, calling handler\n", r.URL.Path)
					sessionHandler.RequestHandler()
					value(sessionHandler)
					sessionHandler.Renderhandler.StartRender()
				} else {
					WriteConsolef("Route not found for URL: %s\n", r.URL.Path)
					http.Error(w, "404 Error : Route not found ", 404)
				}
			} else {
				WriteConsole("Failed to start session")
				return
			}
		}
	}
}
