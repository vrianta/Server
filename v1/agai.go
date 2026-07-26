package agai

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/vrianta/agai/v1/config"
	"github.com/vrianta/agai/v1/database"
	"github.com/vrianta/agai/v1/internal/flags"
	"github.com/vrianta/agai/v1/internal/session"
	"github.com/vrianta/agai/v1/internal/template"
	"github.com/vrianta/agai/v1/model"
	"github.com/vrianta/agai/v1/utils"
	log "github.com/vrianta/gulog"
)

// Global instance of the server
type Mod struct{}
type modInterface interface {
	Run(controllerInterface)
}

func (m *Mod) Run(*Controller) {
	log.Error("Mod is not settedup properly, it is still calling the default Run Function.")
	os.Exit(-102)
}

var (
	routerHandler   = Handler
	middlewareFuncs []func()
	modsStorage     []modInterface
)

type app struct {
	*http.Server
	routeTable routes
}

var agai = app{
	routeTable: make(routes),
}

/*
 * File to handle Arguments from the user
 * storing all the argument flags
 */

func init() {
	// go through all the arugments and enable some flags
	if len(os.Args) < 2 {
		print_help()
		os.Exit(0)
		return
	}
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--migrate-model", "-mm":
			flags.SyncDatabaseEnabled = true
		case "--migrate-component", "-mc":
			flags.SyncComponentsEnabled = true
		case "--start-server", "-ss":
			flags.StartServer = true
		case "--show-dsn", "-sdn":
			flags.ShowDsn = true
		case "--help", "-h":
			print_help()
			os.Exit(1)

		default:
			println("Wrong Argument Passed plesae use go run . --help/-h to get the list of arguments")
			os.Exit(1)
		}
	}
}

func New() *app {
	return &agai
}

func print_help() {
	fmt.Println("Flags:")
	fmt.Println("  --migrate-model,     -mm   Run model database migrations")
	fmt.Println("  --migrate-component, -mc   Sync components with the database")
	fmt.Println("  --start-server,      -ss   Start the HTTP server")
	fmt.Println("  --show-dns,          -sdn  Show Dsn if the database connnection failed")
	fmt.Println("  --help,              -h    Show this help message")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  go run . --migrate-model --start-server")
}

// waitForPort waits until the port is available or timeout occurs
func waitForPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			return nil // port is free
		}
		log.WriteLogf("[WARN]: Port %s is busy, waiting...\n", addr)
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("port %s did not become free in time", addr)
}

// Start runs the HTTP server
func (s *app) Start() {

	database.Init()
	s.setup_static_folders()
	s.setup_css_folder()
	s.setup_js_folder()

	model.Init() // intialsing model with creating tables and updating them

	// Starting Session Handler to Manage Session Expiry
	go session.StartSessionHandler()
	go session.StartLRUHandler() // Start the LRU handler for session management

	// setting up the Custom Routing Handler for the syste
	// http.HandleFunc("/", routerHandler)

	// Define the server configuration
	addr := config.GetWebConfig().Host + ":" + config.GetWebConfig().Port
	s.Server = &http.Server{
		Addr: addr,
	}

	// Settings Password Cost in Utils Package
	utils.PasswordCost = config.GetWebConfig().PassordCost

	// Wait for port to be free before starting
	if err := waitForPort(addr, 20*time.Second); err != nil {
		panic("[Server] " + err.Error())
	}

	if _, ok := template.GetTemplate("404"); !ok {
		http.HandleFunc("/404/", func(w http.ResponseWriter, r *http.Request) {
			w.Write(_404__)

		})
		http.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/404/", int(HttpStatus.SeeOther))

		})
	}

	// creating default route /
	if !rootRegistered {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/404/", int(HttpStatus.SeeOther))
		})
	}

	if flags.StartServer {
		if config.GetWebConfig().Host == "" {
			log.WriteLogf("[Server] Started at : http://localhost:%s\n", config.GetWebConfig().Port)
		} else {
			log.WriteLogf("[Server] Started at : http://%s\n", addr)
		}

		fmt.Print("---------------------------------------------------------\n\n")

		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic("[Server] Failed to start: " + err.Error())
		}
	}
}

// if the user want to have a custom Routing Handler the they can use this function to register it
func (s *app) RegisterCustomRoutingHandler(_func func(w http.ResponseWriter, r *http.Request)) {
	routerHandler = _func
}

func (s *app) setup_static_folders() {
	// Create a file server handler
	for _, folder := range config.GetWebConfig().StaticFolders {
		fs := http.FileServer(http.Dir(folder))
		http.Handle("/"+folder+"/", http.StripPrefix("/"+folder+"/", fs))
	}
}

// Generating Creating Routes for the Css Folders
func (s *app) setup_css_folder() {
	for _, folder := range config.GetWebConfig().CssFolders {
		http.HandleFunc("/"+folder+"/", staticFileHandler("text/css; charset=utf-8"))
	}
}

func (s *app) setup_js_folder() {
	for _, folder := range config.GetWebConfig().JsFolders {
		http.HandleFunc("/"+folder+"/", staticFileHandler("application/javascript; charset=utf-8"))
	}
}

// function to add middleware for the application to run on each request
// Function calling will be sequencial
func (a *app) Use(middleware func()) {
	middlewareFuncs = append(middlewareFuncs, middleware)
}

// function to register mods - make sure the mod will run before any middle wares and before the request is handled by the controller
func (a *app) RegisterMod(mod modInterface) {
	modsStorage = append(modsStorage, mod)
}
