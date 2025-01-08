# Go Server Package

This Go server package provides an easy way to create and start a simple web server with configurable routes and session management.

## Features

- **Configurable Routes**: Map URLs to handler functions easily.
- **Session Handling**: Maintain and remove session handlers.
- **Simple Server Setup**: Create a server and start it with a few lines of code.
- **Customizable**: Extend the server with additional middleware, handlers, or logic.

## Installation

To use the server package, first import it into your Go project.

```go
import "path/to/your/server/package"
```

## Usage

### Step 1: Create a new server instance

You can create a new server by calling the `New` function with the desired host, port, and routes.

```go
_s := server.New("", "8080", server.ROUTETYPE{
    "/":               src.Home,
    "/get-token":      token.GetToken,
    "/validate-token": token.ValidateToken,
    "/logout":         login.Logout,
    "/register":       register.RegisterUser,
    "/create-event":   event.Create,
    "/update-user":    update.User,
    "/get-contents":   src.Get,
    "/apply-to-event": event.Apply,
    "/get-applied-events":  event.GetRegisteredEvents,
    "/withdraw-from-event": event.WithdrawFromEvent,
})
```

### Step 2: Start the server

Once the server is initialized, call the `StartServer` method to start it.

```go
_s.StartServer()
```

### Step 3: Session Management

You can manage sessions by using the `SessionHandler` map. You can remove a session by calling the `RemoveSessionHandler` function.

```go
server.RemoveSessionHandler(&sessionID)
```

## Code Explanation

### `ServerHandlerType`

This struct holds the server configuration, including the host, port, routes, and session handlers.

```go
type ServerHandlerType struct {
    Host string
    Port string
    Routes ROUTETYPE
    SessionHandler map[string]SessionHandler
}
```

### `New(host, port, routes)`

The `New` function creates a new server instance with the provided configuration.

```go
func New(host, port string, routes ROUTETYPE) *ServerHandlerType
```

### `StartServer()`

The `StartServer` method starts the HTTP server with the configured routes and host/port.

```go
func (sh *ServerHandlerType) StartServer() error
```

### `RemoveSessionHandler(sessionID *string)`

This function removes the session from the `SessionHandler` map.

```go
func RemoveSessionHandler(sessionID *string)
```

## Example

Here's an example of how to set up and start a server:

```go
package main

import (
    "fmt"
    "path/to/your/server/package"
    "path/to/your/handlers"
)

func main() {
    serverInstance := server.New("localhost", "8080", server.ROUTETYPE{
        "/":              handlers.Home,
        "/get-token":     handlers.GetToken,
        "/validate-token": handlers.ValidateToken,
        "/logout":        handlers.Logout,
        "/register":      handlers.RegisterUser,
    })

    err := serverInstance.StartServer()
    if err != nil {
        fmt.Println("Server failed to start:", err)
    }
}
```

## Demo: Get Function

You can define functions to interact with the session handler. Here is a demo of a `Get` function that takes a `SessionHandler` as an argument.

```go
func Get(sessionHandler *server.SessionHandler) {
    // Example of accessing session data
    fmt.Println("Session Handler:", sessionHandler)
    // You can add logic to handle session data here
}
```
