package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"

	"redis_golang/internal/metrics"
	"redis_golang/internal/storage/memory"
	"redis_golang/pkg/logger"
	"redis_golang/ui/web"
)

func RunHTTPServer(port int) {
	mux := http.NewServeMux()

	// API: Stats
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		keys := memory.GetAllKeys() // NOTE: O(N) operation, fine for now but optimize later
		
		stats := map[string]interface{}{
			"connections":    metrics.GetActiveConnections(),
			"total_commands": metrics.GetTotalCommands(),
			"hits":           metrics.GetCacheHits(),
			"misses":         metrics.GetCacheMisses(),
			"total_keys":     len(keys),
		}
		json.NewEncoder(w).Encode(stats)
	})

	// API: Logs
	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		logs := logger.MemHandler.FormatLogs()
		json.NewEncoder(w).Encode(map[string]interface{}{"logs": logs})
	})

	// API: Keys
	mux.HandleFunc("/api/keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		keys := memory.GetAllKeys()
		json.NewEncoder(w).Encode(map[string]interface{}{"keys": keys})
	})

	// Serve Static Files
	// Go 1.16+ embed.FS
	staticFS, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		logger.Log.Error("Failed to load static web assets", "error", err)
		return
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	logger.Log.Info("Starting Web Dashboard", "url", "http://"+addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Log.Error("HTTP Server failed", "error", err)
	}
}
