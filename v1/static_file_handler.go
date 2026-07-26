package agai

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/vrianta/agai/v1/utils"
	log "github.com/vrianta/gulog"
)

type (
	fileCacheEntry struct {
		Uri          string    // path of the template file
		LastModified time.Time // date when the file last modified
		Data         string    // template data of the file before modified
	}
)

var fileCache sync.Map // map[string]FileInfo

// StaticFileHandler serves static files with caching support.
// It checks if the file exists in the cache and serves it directly if the cache is valid.
// Otherwise, it reads the file from disk, caches it, and serves it.
// Parameters:
// - contentType: The MIME type of the file being served.
// Returns:
// - http.HandlerFunc: A handler function for serving static files.
func staticFileHandler(contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_filePath := "." + r.URL.Path

		// Attempt to load from cache
		val, _ := fileCache.Load(_filePath)
		cached, ok := val.(fileCacheEntry)

		info, err := os.Stat(_filePath)
		if err != nil {
			log.WriteLogf("%s", err.Error())
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", contentType)

		// If cached data exists and mod time matches then serve from cache
		if ok && cached.LastModified.Equal(info.ModTime()) {
			w.Write([]byte(cached.Data))
			return
		}

		// Read file from disk and cache it
		_fileData := utils.ReadFromFile(_filePath)
		newRecord := fileCacheEntry{
			Uri:          _filePath,
			LastModified: info.ModTime(),
			Data:         string(_fileData),
		}
		fileCache.Store(_filePath, newRecord)
		w.Write([]byte(_fileData))
	}
}
