package server

import (
	"fmt"
	"net/http"
)

type ROUTETYPE map[string]func(*SessionHandler)
type ServerHandler struct {
	Host string
	Port string

	Routes ROUTETYPE

	SessionHandler map[string]SessionHandler
}

var (
	serverHandler *ServerHandler
)

func New(host, port string, routes ROUTETYPE) *ServerHandler {
	serverHandler = &ServerHandler{
		Host:   host,
		Port:   port,
		Routes: routes,

		SessionHandler: make(map[string]SessionHandler),
	}
	return serverHandler
}

func (sh *ServerHandler) StartServer() error {

	http.HandleFunc("/", sh.routingHandler)

	// Define the server configuration
	server := &http.Server{
		Addr: sh.Host + ":" + sh.Port, // Host and port
	}

	// Log("Server is running on http://" + Host + ":" + Port)

	// Start the server
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting server:", err)
	} else {
		return err
	}
	return nil
}

func RemoveSessionHandler(sessionID *string) {
	delete(serverHandler.SessionHandler, *sessionID)
}
