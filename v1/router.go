package agai

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	log "github.com/vrianta/gulog"
)

// it will keep a record if the user has registered / root path or not
var rootRegistered bool = false
var rootPath string = ""

/*
 * If you want to change the initial path for next registered paths then pass paths
 * example you want next all paths to be /example/then-other-paths then pass example in the function
 * GroupPathWith("example")
 * As is you can pass multiple paths to be initial path
 * GroupPathWith("example", "path1") -> /example/path1
 * CreateRoute[Controller]("login") -> /example/path1/login/
 */
func GroupNextPathsWith(route ...string) {
	rootPath = "/" + strings.Join(route, "/")
}

/*
 * CreateRoute("login", "register")
 * it will create two paths
 * 1. /login/register/
 * 2. /login/register
 */
func CreateRoute[T any, PT interface {
	*T
	controllerInterface
}](route ...string) {

	if route[0] == "" {
		panic("Empty Route not allowed")
	}

	if len(route) > 1 && route[0] == "/" {
		panic("Multiple Route Registration with / not allowed")
	}

	// for root route registration
	if route[0] == "/" && rootPath == "" { // for the inital route because we can not have more than one /
		if route[0] == "/" && rootPath == "" {
			rootRegistered = true
		}

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.Redirect(w, r, "/404/", int(HttpStatus.SeeOther))
				return
			}
			var tempController PT = new(T)
			tempController.init(w, r)
			runRequest(w, r, tempController)
		})
		return
	}

	// eg: root path set as /admin and you creating a route for / then it will create a route for /admin/ and /admin
	if (route[0] == "/") && rootPath != "" {
		fr := rootPath + "/" // final route
		http.HandleFunc(fr, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != fr {
				http.Redirect(w, r, "/404/", int(HttpStatus.SeeOther))
				return
			}
			var tempController PT = new(T)
			tempController.init(w, r)
			runRequest(w, r, tempController)
		})

		http.HandleFunc(rootPath, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, fr, int(HttpStatus.SeeOther))
		})
		return
	} else {

		redirected_route := rootPath + "/" + strings.Join(route, "/")
		actual_route := rootPath + "/" + strings.Join(route, "/") + "/"
		http.HandleFunc(actual_route, func(w http.ResponseWriter, r *http.Request) {
			var tempController PT = new(T)
			tempController.init(w, r)
			runRequest(w, r, tempController)
		})

		http.HandleFunc(redirected_route, func(w http.ResponseWriter, r *http.Request) {
			log.Info("Redirecting to: %s\n", actual_route)
			http.Redirect(w, r, actual_route, int(HttpStatus.SeeOther))
		})

		log.Info("[Route] \nRegistered route: %s\nRedirected Route: %s", redirected_route, actual_route)
	}

}

// TODO : in Future
// // Function to create Route with getting function as argument
// last route is the function name
func CreateRouteWithFunc[T any, PT interface {
	*T
	controllerInterface
}](handler func(PT) View) {

	method_name := getFunctionName(handler)

	redirected_route := rootPath + "/" + method_name
	actual_route := rootPath + "/" + method_name + "/"

	// printMethods(new(T))

	http.HandleFunc(actual_route, func(w http.ResponseWriter, r *http.Request) {
		// copy the controller
		var tempController PT = new(T)
		tempController.init(w, r)

		vfunc := func() View {
			return handler(tempController)
		}

		runRequestForFunction(w, tempController, vfunc)
	})

	http.HandleFunc(redirected_route, func(w http.ResponseWriter, r *http.Request) {
		log.Info("Redirecting to Safer Path: %s\n", actual_route)
		http.Redirect(w, r, actual_route, int(HttpStatus.SeeOther))
	})

	log.Info("[Route] \nRegistered route: %s\nRedirected Route: %s", redirected_route, actual_route)

}

func CallMethodByName(obj any, methodName string, args ...interface{}) (func() View, error) {
	v := reflect.ValueOf(obj)
	// printMethods(obj)
	method := v.MethodByName(methodName)

	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	// Convert args to reflect.Value
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	return method.Interface().(func() View), nil
}

func getFunctionName(i any) string {

	t := strings.Split(runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name(), ".")
	return t[len(t)-1]
}
