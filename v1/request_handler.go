package agai

import (
	"bytes"
	"net/http"
	"sync"

	"github.com/vrianta/agai/v1/config"
	"github.com/vrianta/agai/v1/internal/template"
	log "github.com/vrianta/gulog"
)

// Handler processes incoming HTTP requests and manages user sessions.
// It checks if the user has an existing session and handles session creation or validation.
// Based on the session and route, it invokes the appropriate controller method.
// Parameters:
// - w: The HTTP response writer.
// - r: The HTTP request.
type (
	controllerInterface interface { // Resembeles Controller Package
		GET() View
		POST() View
		PUT() View
		DELETE() View
		PATCH() View
		HEAD() View
		OPTIONS() View
		init(w http.ResponseWriter, r *http.Request)
		IsLoggedIn() bool
	}
	routes map[string]func() controllerInterface
)

var template_bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func Handler(w http.ResponseWriter, r *http.Request) {

	if _c, found := agai.routeTable[r.URL.Path]; found {
		tempController := _c()
		tempController.init(w, r)
		runRequest(w, r, tempController)
	} else {
		http.Error(w, "404 Error : Route not found ", http.StatusNotFound)
	}
}

func runRequest(w http.ResponseWriter, r *http.Request, c controllerInterface) {

	execute_template := func(view *view) {
		__template, ok := template.GetTemplate(view.name)

		if !ok {
			log.Error("No Template Available as such name %s", view.name)
			return
		}

		if !config.GetWebConfig().Build {
			// log.WriteLogff("Updating the Template")
			__template.Update()
		}
		if err := executeTemplate(w, __template, view.response); err != nil {
			log.Error("Error rendering template: %T\n", err)
			panic(err.Error())
		}
	}

	executeView := func(view *view) {
		if view == nil {
			return
		}
		if view.asJson {
			// user want the response to be send as json
			w.Write(view.ToJson())
			return
		}
		execute_template(view)
	}
	c.IsLoggedIn()
	switch r.Method {
	case "GET":
		if vfunc := c.GET(); vfunc != nil {
			executeView(vfunc()) // GET method of controller returns a view
		}

	case "POST":
		if vfunc := c.POST(); vfunc != nil {
			executeView(vfunc()) // GET method of controller returns a view
		}

	case "DELETE":
		if vfunc := c.DELETE(); vfunc != nil {
			executeView(vfunc()) // GET method of controller returns a view
		}

	case "PATCH":
		if vfunc := c.PATCH(); vfunc != nil {
			executeView(vfunc()) // GET method of controller returns a view
		}

	case "PUT":
		if vfunc := c.PUT(); vfunc != nil {
			executeView(vfunc()) // GET method of controller returns a view
		}

	case "HEAD":
		if vfunc := c.HEAD(); vfunc != nil {
			executeView(vfunc()) // GET method of controller returns a view
		}
	case "OPTIONS":
		if vfunc := c.OPTIONS(); vfunc != nil {
			executeView(vfunc()) // GET method of controller returns a view
		}
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func runRequestForFunction(w http.ResponseWriter, c controllerInterface, defFunc func() View) {

	execute_template := func(view *view) {
		__template, ok := template.GetTemplate(view.name)

		if !ok {
			log.Error("No Template Available as such name %s", view.name)
			return
		}

		if !config.GetWebConfig().Build {
			// log.WriteLogff("Updating the Template")
			__template.Update()
		}
		if err := executeTemplate(w, __template, view.response); err != nil {
			log.Error("Error rendering template: %T\n", err)
			panic(err.Error())
		}
	}

	executeView := func(view *view) {
		if view == nil {
			return
		}
		if view.asJson {
			// user want the response to be send as json
			w.Write(view.ToJson())
			return
		}
		execute_template(view)
	}
	c.IsLoggedIn()

	log.WriteLogf("Getting Default Method")
	if vfunc := defFunc(); vfunc != nil {
		executeView(vfunc()) // GET method of controller returns a view
	}
}

func executeTemplate(w http.ResponseWriter, _template *template.Context, __response any) error {
	// Use buffer pool for rendering
	buf := template_bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer template_bufPool.Put(buf)

	switch _template.ViewType {
	case template.ViewTypes.PhpTemplate:
		if _template.Php != nil {
			if err := _template.Php.Execute(buf, __response); err != nil {
				return err
			}
		} else {
			panic("php Template is not registered")
		}
	case template.ViewTypes.HtmlTemplate:
		if _template.Html != nil {
			if err := _template.Html.Execute(buf, __response); err != nil {
				return err
			}
		}
	default:
		if _template.Html != nil {
			if err := _template.Html.Execute(buf, __response); err != nil {
				return err
			}
		}
	}

	w.Write(buf.Bytes())
	return nil
}
