// Mock Cronitor API server for e2e testing
// Captures all requests and allows verification via /debug endpoints
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CapturedRequest stores details of an incoming request
type CapturedRequest struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	QueryParams map[string]string `json:"query_params"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	Timestamp   time.Time         `json:"timestamp"`
}

// RequestStore thread-safe storage for captured requests
type RequestStore struct {
	mu                sync.RWMutex
	monitorRequests   []CapturedRequest
	telemetryRequests []CapturedRequest
}

var store = &RequestStore{}

func main() {
	// Monitors API endpoint (PUT /api/monitors)
	http.HandleFunc("/api/monitors", handleMonitors)

	// Telemetry API endpoint (POST /ping/...)
	http.HandleFunc("/ping/", handleTelemetry)

	// Debug endpoints to query captured requests
	http.HandleFunc("/debug/monitors", handleDebugMonitors)
	http.HandleFunc("/debug/telemetry", handleDebugTelemetry)
	http.HandleFunc("/debug/clear", handleDebugClear)
	http.HandleFunc("/debug/health", handleHealth)

	// Also handle /p/ prefix for telemetry (alternative format)
	http.HandleFunc("/p/", handleTelemetry)

	log.Println("Mock Cronitor API server starting on :8080")
	log.Println("Endpoints:")
	log.Println("  PUT  /api/monitors     - Captures monitor sync requests")
	log.Println("  POST /ping/{key}/{id}  - Captures telemetry requests")
	log.Println("  GET  /debug/monitors   - Returns captured monitor requests")
	log.Println("  GET  /debug/telemetry  - Returns captured telemetry requests")
	log.Println("  POST /debug/clear      - Clears all captured requests")
	log.Println("  GET  /debug/health     - Health check")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func captureRequest(r *http.Request) CapturedRequest {
	// Read body
	body, _ := io.ReadAll(r.Body)

	// Capture query params
	queryParams := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	// Capture relevant headers
	headers := make(map[string]string)
	for _, h := range []string{"Content-Type", "User-Agent", "Authorization"} {
		if v := r.Header.Get(h); v != "" {
			headers[h] = v
		}
	}

	return CapturedRequest{
		Method:      r.Method,
		Path:        r.URL.Path,
		QueryParams: queryParams,
		Headers:     headers,
		Body:        string(body),
		Timestamp:   time.Now(),
	}
}

func handleMonitors(w http.ResponseWriter, r *http.Request) {
	req := captureRequest(r)

	store.mu.Lock()
	store.monitorRequests = append(store.monitorRequests, req)
	store.mu.Unlock()

	log.Printf("Captured monitor request: %s %s", req.Method, req.Path)
	if req.Body != "" {
		// Pretty print the body for logging
		var prettyBody interface{}
		if err := json.Unmarshal([]byte(req.Body), &prettyBody); err == nil {
			prettyJSON, _ := json.MarshalIndent(prettyBody, "  ", "  ")
			log.Printf("  Body: %s", string(prettyJSON))
		}
	}

	// Return a mock response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Return the monitors that were sent (echo back for verification)
	if req.Body != "" {
		w.Write([]byte(req.Body))
	} else {
		w.Write([]byte("[]"))
	}
}

func handleTelemetry(w http.ResponseWriter, r *http.Request) {
	req := captureRequest(r)

	store.mu.Lock()
	store.telemetryRequests = append(store.telemetryRequests, req)
	store.mu.Unlock()

	// Extract monitor key from path
	parts := strings.Split(req.Path, "/")
	monitorKey := ""
	if len(parts) >= 4 {
		monitorKey = parts[3] // /ping/{api_key}/{monitor_key}
	}

	log.Printf("Captured telemetry request: %s %s (monitor: %s, state: %s, env: %s)",
		req.Method, req.Path, monitorKey,
		req.QueryParams["state"], req.QueryParams["env"])

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleDebugMonitors(w http.ResponseWriter, r *http.Request) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":    len(store.monitorRequests),
		"requests": store.monitorRequests,
	})
}

func handleDebugTelemetry(w http.ResponseWriter, r *http.Request) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":    len(store.telemetryRequests),
		"requests": store.telemetryRequests,
	})
}

func handleDebugClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	store.mu.Lock()
	store.monitorRequests = nil
	store.telemetryRequests = nil
	store.mu.Unlock()

	log.Println("Cleared all captured requests")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Cleared"))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"healthy","monitor_count":%d,"telemetry_count":%d}`,
		len(store.monitorRequests), len(store.telemetryRequests))
}
