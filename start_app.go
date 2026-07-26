package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/vrianta/agai/v1/config"
	log "github.com/vrianta/gulog"
)

// var clientChan chan string = make(chan string)
var mu sync.Mutex
var clients = make(map[chan string]bool)
var run_app *exec.Cmd
var migrate_models *exec.Cmd
var migrate_components *exec.Cmd
var browser_proc *exec.Cmd
var output_app_name string
var wait_for_hot_reloader sync.WaitGroup

// this falag is needed because after we restart the application on componenent file change it again pdate the components so we have to detect
// if we are not detecting the component after a application start or an application restart
var FLAG_restarted_application_after_component_change bool = false

func init() {
	if runtime.GOOS == "windows" {
		output_app_name = "app.exe"
	} else {
		output_app_name = "app"
	}

	// run_app = runAppCmd()
	migrate_models = new_migrate_model_cmd()
	migrate_components = new_migrate_component_cmd()
	// config.init()
}

// ---- helpers ----

func streamLogs(r io.ReadCloser, prefix string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Info("%s %s", prefix, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Error("%s scanner error: %v", prefix, err)
	}
}

// Start command asynchronously and stream its output
func startServer(cmd *exec.Cmd) error {
	// 1) pipes BEFORE Start()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	// 2) start
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	// 3) stream logs
	go streamLogs(stdout, "")
	go streamLogs(stderr, "")
	return nil
}

// Kill old server (if any), wait (reap), create new cmd, start + stream
func restartApp() error {

	if err := run_app.Process.Kill(); err != nil {
		log.Error("Failed to Kill the Server Process while restarting the server: %s", err.Error())
	}
	if err := run_app.Wait(); err != nil {
		log.Error("Failed to wait for the all process kill: %s", err.Error())
	} // reap

	run_app = nil
	run_app = runAppCmd()

	FLAG_restarted_application_after_component_change = false

	return startServer(run_app)
}

// Run a short-lived command and capture its combined output
func runAndLog(cmd *exec.Cmd, what string) {
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("%s failed: %v\n%s", what, err, string(out))
		return
	}
	log.Info("%s ok:\n%s", what, string(out))
}

func start_app() {
	if !f.start_app {
		return
	}

	log.Info("Starting server process...")

	run_app = runAppCmd() //. creating the the vinary and command
	// NEW: don't Start() first—use startServer which wires pipes then starts
	if err := startServer(run_app); err != nil {
		panic("Failed to Start the Server: " + err.Error())
	}

	go start_hot_reload()
	go WatchFolders(5 * time.Second)

	url := "http://" + func() string {
		if config.GetHost() == "" {
			return "localhost"
		}
		return config.GetHost()
	}() + ":" + config.GetPort()

	log.Info("Opening browser at: %s", url)
	fmt.Println("Browser URL:", url)
	if err := openBrowser(url); err != nil {
		log.Error("Failed to open browser: %s", err.Error())
	}

	defer func() {
		if run_app != nil && run_app.Process != nil {
			log.Info("Killing server process...")
			_ = run_app.Process.Kill()
			_ = run_app.Wait()
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Info("Running. Press Ctrl+C to exit.")

	<-sigChan

	broadcast("close") // clossing all the tabs which are running the app
	wait_for_hot_reloader.Wait()

	fmt.Println("Interrupt received. Exiting...")
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		// The "" is a placeholder for the window title
		cmd = "cmd"
		args = []string{"/c", "start", "", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	browser_proc = exec.Command(cmd, args...)
	return browser_proc.Start()
}

func start_hot_reload() {
	log.Info("Starting hot reload server on port 8888")
	http.HandleFunc("/hot-reload", sseHandler)
	err := http.ListenAndServe(":8888", nil)
	if err != nil {
		log.Error("Hot reload server failed: %s", err.Error())
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("SSE client connected")

	// SSE headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Each client gets its own channel
	client := make(chan string)

	mu.Lock()
	clients[client] = true
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(clients, client)
		mu.Unlock()
		close(client)
		log.Debug("SSE client disconnected")
	}()

	// Listen and send events
	for {
		select {
		case msg, ok := <-client:
			if !ok {
				return
			}
			if msg == "close" {
				fmt.Println("clossing tabs")
				wait_for_hot_reloader.Done()
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return // Client closed connection
		}
	}
}

func broadcast(msg string) {
	mu.Lock()
	defer mu.Unlock()

	for ch := range clients {
		select {
		case ch <- msg:
			if msg == "close" {
				wait_for_hot_reloader.Add(1)
			}

		default:
			// Drop slow or dead clients
			close(ch)
			delete(clients, ch)
		}
	}
}

func WatchFolders(interval time.Duration) {
	log.Info("Starting file watcher...")

	modTimestamps := make(map[string]time.Time)
	// componentTimestamps := make(map[string]time.Time)
	generalTimestamps := make(map[string]time.Time)

	for {
		moduleChanged := checkDir("models", modTimestamps)
		// componentChanged := checkDir("components", componentTimestamps)
		generalChanged := checkGeneral(".", generalTimestamps)

		if moduleChanged {
			log.Info("Model file changed")
			onModuleChange()
			// time.Sleep(interval)
			broadcast("reload")
		}
		// if componentChanged {
		// 	log.Info("Component files changed")
		// 	onComponentChange()
		// 	time.Sleep(interval)
		// 	// log.Warn("We Deteceted a Component Change Please -restart the application so | curretnly hot reload is not supported for components")
		// 	broadcast("reload")
		// }
		if generalChanged {
			log.Info("General files changed")
			onGeneralChange()
			// time.Sleep(interval)
			broadcast("reload")
		}

	}
}

func checkDir(root string, lastMod map[string]time.Time) bool {
	changed := false

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		modTime := info.ModTime()

		// If we've never seen this file before, add it to the map but don't trigger change
		if _, ok := lastMod[path]; !ok {
			lastMod[path] = modTime
			return nil
		}

		if modTime.After(lastMod[path]) {
			log.Debug("Change detected in %s", path)
			lastMod[path] = modTime
			changed = true
		}
		return nil
	})

	if err != nil {
		log.Error("Error watching %s: %s", root, err.Error())
	}

	return changed
}

/*
 * Checking general files which are going to created by the user or not really important to do extra lookup
 * return a bool to show if the system has to restart the server or not
 */
func checkGeneral(root string, lastMod map[string]time.Time) bool {
	changed := false

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip specific directories
		if info.IsDir() && (info.Name() == "models" || info.Name() == "components" || info.Name() == ".git") {
			return filepath.SkipDir
		}
		if info.Name() == "sessions.data" || info.Name() == output_app_name {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		modTime := info.ModTime()

		// Don't trigger change on first encounter
		if _, ok := lastMod[path]; !ok {
			lastMod[path] = modTime
			return nil
		}

		if modTime.After(lastMod[path]) {
			log.Debug("General change detected in %s", path)
			lastMod[path] = modTime
			changed = true
		}

		return nil
	})

	if err != nil {
		log.Error("Error watching general files: %s", err.Error())
	}

	return changed
}

func app_build() {
	if err := exec.Command("go", "build", "-o", output_app_name, ".").Run(); err != nil {
		log.Error("Failed to Build the app - %s", err.Error())
	}
}

var currentDir, _ = os.Getwd()
var runBName string = filepath.Base(currentDir) + "run_" + "a7Kp9LmX2QzN8rTf"

/*
 * Run the solution
 * Command it is running is go run .
 * Returns a exec.Cmd which can be used to run the application
 */
func runAppCmd() *exec.Cmd {
	buildAgai() // building the application to run
	r := exec.Command(runBName, "-ss")

	return r
}

// Clean the build bindary
func buildAgai() {
	if err := exec.Command("go", "build", ".", "-o", runBName).Run(); err != nil {
		panic("Failed to  Build app " + err.Error())
	}
}

func new_app_cmd() *exec.Cmd {
	app_build()
	r := exec.Command("./"+output_app_name, "-ss")
	r.SysProcAttr = &syscall.SysProcAttr{
		// CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP, // Windows only
	}
	return r
}

func new_migrate_model_cmd() *exec.Cmd {
	return exec.Command("go", "run", ".", "-mm")
}

func new_migrate_component_cmd() *exec.Cmd {
	return exec.Command("go", "run", ".", "-mc")
}

func onModuleChange() {
	log.Info("Restarting server due to module changes...")

	// migrate models (short-lived)
	migrate_models = new_migrate_model_cmd()
	runAndLog(migrate_models, "Model migration")

	if err := restartApp(); err != nil {
		log.Error("Failed to restart server: %v", err)
		return
	}
	log.Info("Server restarted after module migration.")
}

func onComponentChange() {
	if !FLAG_restarted_application_after_component_change {
		FLAG_restarted_application_after_component_change = true
		return
	}

	log.Info("Restarting server due to component changes...")

	migrate_components = new_migrate_component_cmd()
	runAndLog(migrate_components, "Component migration")

	if err := restartApp(); err != nil {
		log.Error("Failed to restart server: %v", err)
		return
	}

	log.Info("Server restarted after component migration.")
	FLAG_restarted_application_after_component_change = false
}

func onGeneralChange() {
	log.Warn("Restarting server due to general changes...")

	if err := restartApp(); err != nil {
		log.Error("Failed to restart server: %v", err)
		return
	}
	log.Info("Server restarted due to general change.")
}
