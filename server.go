package server

import (
	"fmt"
	"net/http"
)

type ROUTETYPE map[string]func(*SessionHandler)
type ServerHandlerType struct {
	Host string
	Port string

	Routes ROUTETYPE

	SessionHandler map[string]SessionHandler
}

var (
	ServerHandler *ServerHandlerType
)

func New(host, port string, routes ROUTETYPE) *ServerHandlerType {
	ServerHandler = &ServerHandlerType{
		Host:   host,
		Port:   port,
		Routes: routes,

		SessionHandler: make(map[string]SessionHandler),
	}
	return ServerHandler
}

func (sh *ServerHandlerType) StartServer() error {

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
	delete(ServerHandler.SessionHandler, *sessionID)
}
