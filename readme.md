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

## Handling Sessions

Sessions are handled through the `SessionHandler` object, which stores session data (such as `uid` and `token`). To access and manipulate session data, use the `VAR` field of `SessionHandler`.

### Example:

```go

if uid != sessionHandler.VAR["uid"].(string) {
    sessionHandler.Renderhandler.Render(server.GetResponse("WORNGUID", "Wrong User Id passed", false))
    return
}

if token != sessionHandler.VAR["token"].(string) {
    sessionHandler.Renderhandler.Render(server.GetResponse("WRONGTOKEN", "Wrong token passed", false))
    return
}

```

## Rendering Responses

To render responses to the user, the `Renderhandler` object is used. It holds the response content and sends it back to the client using the `StartRender()` function.

### Example:

```go

sessionHandler.Renderhandler.Render("Hello World")

```

This will send a response with the message "Did not receive the request on POST method" if a non-POST request is made.

## Using POST and GET Data

To handle POST and GET data, use the `sessionHandler.POST` for POST data and `sessionHandler.GET` for GET data. You can also access session data through the `sessionHandler.VAR` field.

### Example:

```go

uid, uidExists := sessionHandler.POST["uid"]
token, tokenExists := sessionHandler.POST["token"]

if !uidExists || !tokenExists {
    missing := []string{}
    if !uidExists {
        missing = append(missing, "UID")
    }
    if !tokenExists {
        missing = append(missing, "token")
    }
    server.WriteConsole(fmt.Sprintf("Missing required fields: %v", missing))
    sessionHandler.Renderhandler.Render(server.GetResponse("MissingCredentials", strings.Join(missing, ", "), false))
    return
}

```

## Demo Example

Here’s a simple demonstration of using the server package to handle routing, sessions, and rendering:

### Example:

```go

func Home(sessionHandler *server.SessionHandler) {
    sessionHandler.Renderhandler.Render(server.GetResponse("CONNECTION", "Connection succeeded successfully", true))
}

func Get(sessionHandler *server.SessionHandler) {
    if !sessionHandler.IsLoggedIn() {
        sessionHandler.Renderhandler.Render(server.GetResponse("NOPOSTMETHOD", "Did not receive the request on POST method", false))
        return
    }

    uid, uidExists := sessionHandler.POST["uid"]
    token, tokenExists := sessionHandler.POST["token"]

    if !uidExists || !tokenExists {
        missing := []string{}
        if !uidExists {
            missing = append(missing, "UID")
        }
        if !tokenExists {
            missing = append(missing, "token")
        }
        server.WriteConsole(fmt.Sprintf("Missing required fields: %v", missing))
        sessionHandler.Renderhandler.Render(server.GetResponse("MissingCredentials", strings.Join(missing, ", "), false))
        return
    }

    if uid != sessionHandler.VAR["uid"].(string) {
        server.WriteConsole("User ID is not matching")
        sessionHandler.Renderhandler.Render(server.GetResponse("WORNGUID", "Wrong User ID passed", false))
        return
    }

    if token != sessionHandler.VAR["token"].(string) {
        server.WriteConsole("Token is not matching")
        sessionHandler.Renderhandler.Render(server.GetResponse("WRONGTOKEN", "Wrong token passed", false))
        return
    }

    // Handle further logic...
}

```

