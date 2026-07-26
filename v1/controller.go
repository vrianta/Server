package agai

/*
Package Controller

This package defines the core Controller struct and methods for the Go Server Framework.
Controllers are responsible for handling HTTP requests, managing session state, and rendering views.

Key Concepts:
-------------
- Each Controller is a struct that maps HTTP methods (GET, POST, etc.) to handler functions.
- The Controller manages its own session, request, and response writer.
- Views are loaded from the configured Views folder, and templates are registered per HTTP method.
- Templates support PHP-like syntax and are rendered using the custom Template engine.
- Session data is accessed via the controller's session field (see Session package).
- The Controller provides helper methods for:
    - Validating configuration (Validate)
    - Registering and executing templates (RegisterTemplate, ExecuteTemplate)
    - Initializing request/response/session objects (InitWR, initSession)
    - Running the correct handler for an HTTP request (runRequest)
    - Copying controller instances (Copy)
    - Accessing the session safely (GetSession)

Usage:
------
- Define a controller struct embedding Controller.Instance.
- Set the View field and handler functions for each HTTP method.
- Register the controller in your router.
- Use the provided methods to manage templates and session state.

See Also:
---------
- Session package: for session management and data storage.
- Template package: for template parsing and rendering.
- Config package: for server and view configuration.

*/

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrianta/agai/v1/internal/session"
	"github.com/vrianta/agai/v1/utils"
	log "github.com/vrianta/gulog"
)

// Routes is a map of HTTP methods to their respective controllers
type (
	Controller struct {
		session *session.Instance // Session object to handle user session

		// privte objects
		W http.ResponseWriter
		R *http.Request

		userInputs map[string]any
	}

	View = func() *view
)

func (c *Controller) init(w http.ResponseWriter, r *http.Request) {
	c.W = w
	c.R = r
}

/*
This file contains local methods for the Controller struct, providing default and internal logic.
*/
func (c *Controller) GET() View {
	response := Response{
		"Massage": "Method Not Implimented",
	}
	return c.ViewAsJson(response)
}

func (c *Controller) POST() View {
	return c.GET()
}

func (c *Controller) PUT() View {
	return c.GET()
}

func (c *Controller) PATCH() View {
	return c.GET()
}

func (c *Controller) DELETE() View {
	return c.GET()
}

func (c *Controller) HEAD() View {
	return c.GET()
}

func (c *Controller) OPTIONS() View {
	return c.GET()
}

/*
 * Store Data in the Session
 */
func (_c *Controller) StoreData(index string, _d any) {
	_c.session.Store(index, _d)
}

/*
 * Get Data From Session Store
 */
func (_c *Controller) GetStoredData(index string) (any, bool) {
	data, ok := _c.session.Data[index]
	return data, ok
}

/*
 * Get Data From Session Store
 */
func (_c *Controller) GetStoredDatas() map[string]any {
	return _c.session.Data
}

// Return all Inputs at once
func (_c *Controller) GetInputs() map[string]any {
	if _c.userInputs == nil {
		_c.parseRequest()
	}
	return _c.userInputs
}

// Return specific input
// if present then value else nil
func (_c *Controller) GetInput(key string) any {
	if _c.userInputs == nil {
		_c.parseRequest()
	}
	if v, ok := _c.userInputs[key]; ok {
		return v
	}
	return nil
}

// func (c *Controller) getToken() (string, error) {
// 	if c.R.Body.Close()
// 	return "", errors.New("no token present")
// }

/*
 * This File is to handle User Inputs
 */
func (_c *Controller) parseRequest() {

	_c.userInputs = make(map[string]any, 30)

	// Log handling of queryBuilder parameters for non-POST methods
	if _c.R.Method == http.MethodGet {
		for key, values := range _c.R.URL.Query() {
			_c.processqueryBuilderParams(key, values)
		}
	}

	contentType := strings.Split(_c.R.Header.Get("Content-Type"), ";")[0]
	switch contentType {
	case "application/json":
		if p, err := io.ReadAll(_c.R.Body); err != nil {
			log.Error("Failed to Read the Joson Data, %v\n", err)
		} else {
			if er := json.Unmarshal(p, &_c.userInputs); er != nil {
				log.Error("Failed to Render the Json Data, %v\n the Json Data: \n %s", er, string(p))
			}
		}
	case "application/x-www-form-urlencoded":
		// Handle form data (application/x-www-form-urlencoded)
		if err := _c.R.ParseForm(); err != nil {
			log.WriteLogf("Error parsing form data | Error - %s\n", err.Error())
		} else {
			for key, values := range _c.R.PostForm {
				_c.processPostParams(key, values)
			}
		}

	case "multipart/form-data":
		// Handle multipart form data (file upload)
		// Note: This case is handled separately below
		if err := _c.R.ParseMultipartForm(10 << 20); err != nil { // 10 MB
			log.WriteLogf("Error parsing multipart form data | Error - %s\n", err.Error())
			return
		}

		// Store uploaded files (keep FileHeader objects so callers can open them)
		if _c.R.MultipartForm != nil {
			for key, fhs := range _c.R.MultipartForm.File {
				if len(fhs) > 1 {
					_c.userInputs[key] = fhs // [](*multipart.FileHeader)
				} else if len(fhs) == 1 {
					_c.userInputs[key] = fhs[0] // *multipart.FileHeader
				}
			}
		}
		for key, values := range _c.R.PostForm {
			_c.processPostParams(key, values)
		}

	default:
		log.Error("Content-Type %s not supported yet raise a issue in github to get it implimented", contentType)
	}

}

// handlequeryBuilderParams processes parameters found in the URL queryBuilder
func (_c *Controller) processqueryBuilderParams(key string, values []string) {
	var err error
	// Check for multiple values

	if len(values) > 1 {
		if _c.userInputs[key], err = utils.JsonToString(values); err != nil {
			// http.Error(sh.W, "Failed to convert data to JSON", http.StatusMethodNotAllowed)
			return
		}
	} else {
		_c.userInputs[key] = values[0] // Store single value as a string
	}
}

// handlePostParams processes parameters found in the POST data
func (_c *Controller) processPostParams(key string, values []string) {
	var err error
	if len(values) > 1 {
		if _c.userInputs[key], err = utils.JsonToString(values); err != nil {
			// http.Error(sh.W, "Failed to convert data to JSON", http.StatusMethodNotAllowed)
			return
		}
	} else {
		_c.userInputs[key] = values[0] // Store single value as a string
	}
}

// File returns the uploaded file header for the given field name.
// If multiple files were uploaded under the same field, File returns the first one.
func (_c *Controller) File(name string) (*multipart.FileHeader, bool) {
	if _c.userInputs == nil {
		_c.parseRequest()
	}

	// FormFile returns (*multipart.FileHeader, error)
	_, fh, err := _c.R.FormFile(name)
	if err != nil {
		log.Error("File not found for field '%s': %v", name, err)
		return nil, false
	}

	return fh, true
}

// SaveFile saves the provided *multipart.FileHeader (as returned by File)
// to destPath. If destPath is a directory (ends with separator or is an
// existing directory) the original filename will be used.
func (_c *Controller) SaveFile(fh *multipart.FileHeader, destPath string) (string, error) {
	if fh == nil {
		return "", fmt.Errorf("nil FileHeader provided")
	}

	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// determine if destPath is a directory
	isDir := strings.HasSuffix(destPath, string(os.PathSeparator))
	if !isDir {
		if fi, err := os.Stat(destPath); err == nil && fi.IsDir() {
			isDir = true
		}
	}

	var outPath string
	if isDir {
		if err := os.MkdirAll(destPath, 0o755); err != nil {
			return "", err
		}
		outPath = filepath.Join(destPath, fh.Filename)
	} else {
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return "", err
		}
		outPath = destPath
	}

	dst, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return outPath, nil
}
