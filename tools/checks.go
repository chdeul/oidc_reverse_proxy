package tools

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type Health struct {
	ready   atomic.Bool
	started time.Time
}

// Liveness: cheap, no dependencies. Just "am I running?"
func (h *Health) liveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status": "alive",
		"uptime": time.Since(h.started).String(),
	})
}

// Readiness: check dependencies. Fail = stop routing traffic.
func (h *Health) readyHandler(w http.ResponseWriter, r *http.Request) {

	if !h.ready.Load() {
		http.Error(w, `{"status":"starting"}`, http.StatusServiceUnavailable)
		return
	}

	//if other check fail {
	//	w.WriteHeader(http.StatusServiceUnavailable)
	//	json.NewEncoder(w).Encode(map[string]string{"status": "db_unavailable"})
	//	return
	//}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func InitChecks() {
	h := &Health{started: time.Now()}
	http.HandleFunc("/healthz/live", h.liveHandler)
	http.HandleFunc("/healthz/ready", h.readyHandler)

	// Mark ready AFTER warmup
	go func() {
		time.Sleep(2 * time.Second) // simulate startup work
		h.ready.Store(true)
	}()
}
