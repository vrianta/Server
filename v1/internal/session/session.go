package session

import (
	"container/heap"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/vrianta/agai/v1/config"
	"github.com/vrianta/agai/v1/utils"
	log "github.com/vrianta/gulog"
)

/*
* Session Management Package
*
* This package provides a robust session management system for handling
* user authentication, session creation, and data persistence.
*
* Features:
* - Session Creation and Management
* - Secure Cookie Handling
* - User Login Management
* - Request Parsing for GET and POST methods
* - Session Expiry Control
*
* Structures:
* - Session: Manages individual user sessions with methods to parse requests,
*   handle cookies, and manage session data.
*
* Functions:
* - NewSession: Initializes a new session with default values.
* - GetSessionID: Retrieves the session ID from cookies.
* - Login: Handles user login by updating session data.
* - IsLoggedIn: Checks if the user is logged in.
* - StartSession: Starts or resumes a session.
* - UpdateSession: Updates the session with new HTTP request/response data.
* - CreateNewSession: Generates a new session ID and assigns it.
* - SetSessionCookie: Adds a session ID cookie to the HTTP response.
* - EndSession: Ends the current session by removing associated data and cookies.
* - ParseRequest: Parses HTTP request parameters for GET and POST data.
* - ProcessqueryBuilderParams: Processes and stores queryBuilder parameters.
* - ProcessPostParams: Processes and stores POST form data.
*
* Usage:
* - Import the package
* - Use `NewSession()` to initialize a session in your handler functions.
* - Call `StartSession()` to begin session tracking.
* - Use `IsLoggedIn()` to verify the user's authentication status.
* - Manage session data directly using the `Store` map.
*
* Example:
*
* func MyHandler(w http.ResponseWriter, r *http.Request) {
*     session := NewSession(w, r)
*     session.StartSession()
*
*     if session.IsLoggedIn() {
*         fmt.Fprintln(w, "Welcome back, ", session.Store["uid"])
*     } else {
*         fmt.Fprintln(w, "Please log in.")
*     }
* }
*
* Best Practices:
* - Ensure that session IDs are securely generated to avoid session fixation attacks.
* - Use HTTPS to encrypt cookies for improved security.
* - Regularly invalidate stale sessions to reduce security risks.
*
* Author: Pritam Dutta
* Date: NA
 */

func init() {
	gob.Register(map[string]*Instance{})
	gob.Register(&Instance{})
	gob.Register(atomic.Int64{})
	heap.Init(&sessionHeap)
	// LoadAllSessionsFromDisk()

	switch config.GetWebConfig().SessionStoreType {
	case "disk", "storage":
		loadAllSessionsFromDisk()
		// case "db", "database":
		// 	if config.GetDatabaseConfig().Host == "" || !database.Initialized {
		// 		panic("You want to use DB as the Session Storage but the Database Is not Initialised please chcek you database connection or the database config")
		// 	}
	}

}

func New(w http.ResponseWriter, r *http.Request) (*Instance, error) {
	if sessionID, err := utils.GenerateSessionID(); err != nil {
		return nil, err
	} else {
		ins := &Instance{
			ID:              sessionID,
			IsAuthenticated: false,
			ExpirationTime:  time.Now().Add(time.Second * 30),
			Data: sessionData{
				"uid": "Guest",
			},
		}
		go Store(ins)

		ins.setCookie(w, r)

		switch config.GetWebConfig().SessionStoreType {
		case session_store_type_database, session_store_type_db:
			data_json, _ := json.Marshal(ins.Data)
			if err := SessionModel.InsertRow(map[string]any{
				"Id":   ins.ID,
				"Data": string(data_json),
			}); err != nil {
				log.Error("Failed to Insert row in the session: %s", err)
			}
		}

		// Wake up the session handler if needed
		select {
		case cleanupTriggerChan <- struct{}{}:
		default:
		}
		return ins, nil
	}
}

func GetSessionID(r *http.Request) (string, error) {
	cookie, err := utils.GetCookie("sessionid", r)
	if cookie != nil {
		return cookie.Value, nil
	}
	return "", err
}

// Function to Remove Session using Session ID in
// the request, if it exists
func RemoveSession(sessionID *string) {
	if sessionID == nil {
		return
	}

	// Lock the mutex for writing
	sessionStoreMutex.Lock()
	delete(instances, (*sessionID))
	sessionStoreMutex.Unlock()

	// delete session f

	// Delete the session from the map

	switch config.GetWebConfig().SessionStoreType {
	case session_store_type_disk, session_store_type_storage:
		go saveAllSessionsToDisk()
	case session_store_type_database, session_store_type_db:
		// delete the sesssion from the database
		go func(_sessionId string) {
			if err := SessionModel.Delete().Where(SessionModel.Fields.Id).Is(_sessionId).Exec(); err != nil {
				log.Error("Failed to Remove session ID from DB : %s", err.Error())
			} else {
				log.Debug("Successfully Deleted the Session")
			}
		}(*sessionID)
	}
}

// Function to Get Session using Session ID in
// the request, if it exists
func Get(sessionID string, w http.ResponseWriter, r *http.Request) (*Instance, bool) {
	if sessionID == "" {
		return nil, false
	}

	sessionStoreMutex.Lock()
	session, exists := instances[sessionID]
	sessionStoreMutex.Unlock()

	if !exists {
		// check if the user chooses to use database for session storage
		// if the session does not existst with the system then will check in the DB
		// by this we can reduce the load on DB
		// TODO : create a session instance and create the session cookies which is importanct becuase working on session storage in Database
		switch config.GetWebConfig().SessionStoreType {
		case session_store_type_db, session_store_type_database:
			db_session, err := SessionModel.Get().Where(SessionModel.Fields.Id).Is(sessionID).First()
			if err != nil {
				log.Error("Failed to get the Session | %s ", err.Error())
				return nil, false
			}
			if db_session != nil {
				id := db_session["Id"]
				data := db_session["Data"]
				data_object := sessionData{}
				json.Unmarshal([]byte(data.(string)), &data_object)
				session = &Instance{
					ID:   id.(string),
					Data: sessionData(data_object),
					// Controller:     make(map[string]ControllerInterface),
					ExpirationTime: time.Now().Add(time.Second * 30),
				}
				log.Debug("Found Session in DB")
				go Store(session)
			} else {
				return nil, false
			}
		}
	} else {
		log.Debug("Found session %s, in the Memory", sessionID)
	}

	go func(sessionID string) {
		// Move to front in LRU
		lruCacheMutex.Lock()
		if elem, ok := lruElementMap[sessionID]; ok {
			lruOrderList.MoveToFront(elem)
		}
		lruCacheMutex.Unlock()
	}(sessionID)

	return session, true
}

func Store(session *Instance) {
	if session == nil {
		return
	}

	if len(instances) >= config.GetWebConfig().MaxSessionCount {
		go evictLRUSession()
	}

	sessionStoreMutex.Lock()
	instances[session.ID] = session
	sessionStoreMutex.Unlock()

	heapAccessMutex.Lock()
	// Push to heap on every store
	heap.Push(&sessionHeap, session)
	heapAccessMutex.Unlock()

	switch config.GetWebConfig().SessionStoreType {
	case "disk", "storage":
		go saveAllSessionsToDisk()
	case "db", "database":
		data_json, _ := json.Marshal(session.Data)
		go func(s *Instance, data string) {
			exists, err := SessionModel.Get().Where(SessionModel.Fields.Id).Is(s.ID).First()
			if err != nil {
				log.Error("Failed to check existing session in DB : %s", err.Error())
				return
			}
			if exists == nil {
				data_json, _ := json.Marshal(session.Data)
				if err := SessionModel.InsertRow(map[string]any{
					"Id":   session.ID,
					"Data": string(data_json),
				}); err != nil {
					log.Error("Failed to Insert row in the session: %s", err)
				}
			} else {
				if json_data, err := json.Marshal(session.Data); err == nil {
					data = string(json_data)
					// Update existing row
					if err := SessionModel.Update(SessionModel.Fields.Data).Where(SessionModel.Fields.Id).Is(session.ID).To(data).Exec(); err != nil {
						log.Error("Failed to Update row in the session: %s", err)
					}
				} else {
					log.Error("Failed to Marshal session data for update: %s", err)
				}

			}
		}(session, string(data_json))
	}

	select {
	case lruOperationChan <- lRUCacheOperation{ID: session.ID, OperationType: "move"}:
	default:
	}
	select {
	case cleanupTriggerChan <- struct{}{}:
	default:
	}
}

func saveAllSessionsToDisk() error {
	// Check if the file exists
	_, err := os.Stat("sessions.data")
	// fileExists := !os.IsNotExist(err)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error checking session file: %w", err)
	}

	// Open file for write (create if not exists, truncate if exists)
	f, err := os.Create("sessions.data")
	if err != nil {
		return fmt.Errorf("failed to create or open session store file: %w", err)
	}
	defer f.Close()

	// Encode the sessions
	enc := gob.NewEncoder(f)

	sessionStoreMutex.RLock()
	defer sessionStoreMutex.RUnlock()

	if err := enc.Encode(instances); err != nil {
		return fmt.Errorf("failed to encode sessions: %w", err)
	}
	return nil
}

func evictLRUSession() {
	// Remove from the back of the list (least recently used)

	elem := lruOrderList.Back()
	if elem == nil {
		return
	}
	sessionID := elem.Value.(string)

	sessionStoreMutex.Lock()
	defer delete(instances, sessionID)
	sessionStoreMutex.Unlock()

	delete(lruElementMap, sessionID)

	switch config.GetWebConfig().SessionStoreType {
	case session_store_type_database, session_store_type_db:
		if err := SessionModel.Delete().Where(SessionModel.Fields.Id).Is(sessionID).Exec(); err != nil {
			log.Error("Failed to delte LRU Session: %s", err.Error())
		}
	}

	lruOrderList.Remove(elem)
}

// Function to keep checking the session expiry to remve them.
// How the function will work is it will loop through instances the current sessions and check when the expiry is time.Time
// Later it find the session which is expired and the one which has the least time to expire.
// It will Expire the expired session and the go on sleep until the least time to expire is reached.
// then it will again go on loop to to repeast the process.
func StartSessionHandler() {
	for {
		heapAccessMutex.Lock()
		if sessionHeap.Len() == 0 {
			heapAccessMutex.Unlock()
			time.Sleep(30 * time.Minute)
			continue
		}

		next := sessionHeap[0] // Peek the earliest

		now := time.Now()
		if next.ExpirationTime.Before(now) {
			RemoveSession(&next.ID)
			sessionHeap.Pop()
			heapAccessMutex.Unlock()
			log.WriteLogf("Session expired: " + next.ID)
			continue
		}

		sleepDuration := time.Until(next.ExpirationTime)
		if sleepDuration < 0 {
			sleepDuration = 0
		}

		select {
		case <-time.After(sleepDuration):
		case <-cleanupTriggerChan:
		}
	}
}

// Function to handle LRU updates in a separate goroutine
// This function listens for updates on the lruOperationChan channel and processes them
// It handles two operations: "move" to move an existing session to the front of the LRU list
// and "InsertRow" to add a new session to the LRU list if it doesn't already exist.
func StartLRUHandler() {
	for update := range lruOperationChan {
		sessionStoreMutex.Lock()
		switch update.OperationType {
		case "move":
			if elem, ok := lruElementMap[update.ID]; ok {
				lruOrderList.MoveToFront(elem)
			}
		case "InsertRow":
			if _, ok := lruElementMap[update.ID]; !ok {
				elem := lruOrderList.PushFront(update.ID)
				lruElementMap[update.ID] = elem
			}
		}
		sessionStoreMutex.Unlock()
	}
}

func (sh *Instance) Login(w http.ResponseWriter, r *http.Request) {
	sh.IsAuthenticated = true
	sh.setCookie(w, r)
	Store(sh)
}

func (sh *Instance) Logout(w http.ResponseWriter, r *http.Request) {
	utils.RemoveCookie("sessionid", w, r)
}

// Sets the session cookie in the client's browser
func (sh *Instance) setCookie(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	sh.ExpirationTime = now.Add(30 * time.Minute).UTC()
	// sh.lastUsed.Store(now.UnixMicro())
	c := &http.Cookie{
		Name:     "sessionid",
		Value:    sh.ID,
		HttpOnly: config.GetWebConfig().Https,
		Expires:  sh.ExpirationTime,
		Path:     "/",
	}

	utils.AddCookie(c, w, r)
}
