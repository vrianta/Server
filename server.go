package Server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

var (
	Host         string
	Port         string
	Routes       = make(map[string]func())
	W            http.ResponseWriter
	R            *http.Request
	POST         = make(map[string]string)
	GET          = make(map[string]string)
	storeMutex   sync.Mutex
	sessionStore = make(map[string]map[string]interface{})
	SESSION      = make(map[string]interface{})

	view      []string
	viewCount int
)

func StartServer() error {

	http.HandleFunc("/", routungHandler)

	// Define the server configuration
	server := &http.Server{
		Addr: Host + ":" + Port, // Host and port
	}

	Log("Server is running on http://" + Host + ":" + Port)

	// Start the server
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting server:", err)
	} else {
		return err
	}
	return nil
}

func routungHandler(w http.ResponseWriter, r *http.Request) {
	W = w
	R = r

	PostHandler()
	GetHandler()
	CreateHeaders()

	if sessionId := GetSessionID(); sessionId != "" {
		storeMutex.Lock()
		SESSION = sessionStore[sessionId]
		storeMutex.Unlock()
	}

	if value, ok := Routes[r.URL.Path]; ok {
		value()
	} else {
		http.Error(W, "404 Error : Route not found ", 404)
	}

	startRender()

	if sessionId := GetSessionID(); sessionId != "" {
		storeMutex.Lock()
		sessionStore[sessionId] = SESSION
		storeMutex.Unlock()
	}
}

func CreateHeaders() {
	// W.Header().Add("X-Session-ID", "sessionID")
}

// getPostData function to read form POST data and return it as a map
func PostHandler() {
	// Ensure the request method is POST
	if R.Method != http.MethodPost {
		// fmt.Println("No POST Method Request Found")
		return
	}

	// Parse multipart form data
	err := R.ParseMultipartForm(10 << 20) // 10 MB limit for file uploads
	if err != nil {
		// fmt.Println(W, "Error parsing multipart form data", http.StatusBadRequest)
		return
	}

	// Loop through all the query parameters
	for key, values := range R.Form {
		// If a parameter has multiple values, store them as a slice of strings
		// For now, we are storing them as strings (pick the first one or join them)
		if len(values) > 1 {
			if POST[key], err = stringArrayToJson(values); err == nil {
				http.Error(W, "Wrong Data Paresed: ", http.StatusMethodNotAllowed)
			}
		} else {
			POST[key] = values[0] // If there's only one value, store it as a string
		}

		// fmt.Println("Setting POST Values - key: ", key, " value: ", POST[key])
	}
}

func GetHandler() {
	// Extract all query parameters from the URL
	queryParams := R.URL.Query()

	// Loop through all the query parameters
	for key, values := range queryParams {
		// If a parameter has multiple values, store them as a slice of strings
		// For now, we are storing them as strings (pick the first one or join them)
		var err error
		if len(values) > 1 {
			if GET[key], err = stringArrayToJson(values); err == nil {
				http.Error(W, "Wrong Data Paresed Not able to convert String to Json ", http.StatusMethodNotAllowed)
			}
		} else {
			GET[key] = values[0] // If there's only one value, store it as a string
		}
		// fmt.Println("Setting POST Values - key: ", key, " value: ", values[0])
	}
}

func File(key string) []byte {
	// Parse multipart form data
	err := R.ParseMultipartForm(10 << 20) // 10 MB limit for file uploads
	if err != nil {
		http.Error(W, "Error parsing multipart form data", http.StatusBadRequest)
		return nil
	}

	// Handle file uploads if present
	if file, _, err := R.FormFile(key); err == nil {
		defer file.Close()
		fileBytes, _ := io.ReadAll(file)
		return fileBytes
	}

	return nil
}

// stringToJson converts a slice of strings to a JSON-encoded string
func stringArrayToJson(data []string) (string, error) {
	// Marshal the slice of strings into a JSON-encoded byte slice
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Return the JSON string (as a string)
	return string(jsonData), nil
}
