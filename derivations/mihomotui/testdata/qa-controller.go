package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const apiAddress, profileAddress = "127.0.0.1:19090", "127.0.0.1:19091"

type controller struct {
	mu      sync.Mutex
	payload string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	apiListener, err := net.Listen("tcp", apiAddress)
	if err != nil {
		log.Fatal(err)
	}
	profileListener, err := net.Listen("tcp", profileAddress)
	if err != nil {
		_ = apiListener.Close()
		log.Fatal(err)
	}
	fixture := &controller{}
	apiServer := &http.Server{Handler: fixture.apiHandler()}
	profileServer := &http.Server{Handler: fixture.profileHandler()}
	errs := make(chan error, 2)
	go serve(errs, func() error { return apiServer.Serve(apiListener) })
	go serve(errs, func() error { return profileServer.Serve(profileListener) })

	fmt.Println("Mihomo API: http://" + apiAddress)
	fmt.Println("Profile URL: http://" + profileAddress + "/profile.yaml")

	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-errs:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("API shutdown: %v", err)
	}
	if err := profileServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("profile shutdown: %v", err)
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		log.Fatal(serveErr)
	}
}

func serve(errs chan<- error, run func() error) { errs <- run() }

func (c *controller) apiHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/configs", c.handleConfigs)
	mux.HandleFunc("/group", handleGroup)
	mux.HandleFunc("/proxies", handleProxies)
	mux.HandleFunc("/group/select/delay", handleGroupDelay)
	mux.HandleFunc("/proxies/", handleProxy)
	mux.HandleFunc("/traffic", handleTraffic)
	mux.HandleFunc("/connections", handleConnections)
	mux.HandleFunc("/logs", handleLogs)
	return mux
}

func (c *controller) profileHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/profile.yaml", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		if r.Header.Get("Authorization") != "" {
			log.Printf("profile request unexpectedly carried Authorization")
			http.Error(w, "profile request carried Authorization", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = fmt.Fprint(w, "mode: rule\nmixed-port: 7890\nproxies: {}\n")
	})
	return mux
}

func (c *controller) handleConfigs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, map[string]any{"mode": "rule", "tun": map[string]bool{"enable": false}})
	case http.MethodPut:
		if r.URL.Query().Get("force") != "true" {
			http.Error(w, "force=true is required", http.StatusBadRequest)
			return
		}
		var update map[string]string
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "invalid config payload", http.StatusBadRequest)
			return
		}
		payload, ok := update["payload"]
		if !ok || len(update) != 1 {
			http.Error(w, "payload must be the sole key", http.StatusBadRequest)
			return
		}
		if strings.Contains(payload, "reject: true") {
			http.Error(w, "fixture rejected payload", http.StatusBadRequest)
			return
		}
		c.mu.Lock()
		c.payload = payload
		c.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w)
	}
}

func handleGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, map[string]any{"proxies": groups()})
}

func handleProxies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, map[string]any{"proxies": map[string]any{
		"select": groups()[0],
		"auto":   groups()[1],
		"node-a": map[string]any{"name": "node-a", "type": "Shadowsocks"},
		"node-b": map[string]any{"name": "node-b", "type": "Shadowsocks"},
	}})
}

func groups() []map[string]any {
	return []map[string]any{
		{"name": "select", "type": "Selector", "now": "node-a", "all": []string{"node-a", "node-b"}},
		{"name": "auto", "type": "URLTest", "now": "node-a", "all": []string{"node-a", "node-b"}},
	}
}

func handleGroupDelay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, map[string]int{"node-a": 25, "node-b": 50})
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/proxies/"), "/delay")
	switch {
	case r.Method == http.MethodPut && r.URL.Path == "/proxies/select":
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/delay") && (name == "node-a" || name == "node-b"):
		delay := 30
		if name == "node-b" {
			delay = 60
		}
		writeJSON(w, map[string]int{"delay": delay})
	default:
		http.NotFound(w, r)
	}
}

func handleTraffic(w http.ResponseWriter, r *http.Request) {
	stream(w, r, func(tick int64) any {
		return map[string]int64{"up": 1024 + tick, "down": 2048 + tick, "upTotal": 4096 + tick, "downTotal": 8192 + tick}
	})
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	stream(w, r, func(tick int64) any {
		return map[string]any{
			"downloadTotal": 8192 + tick,
			"uploadTotal":   4096 + tick,
			"connections": []map[string]any{{
				"id":       "fixture-connection",
				"upload":   512 + tick,
				"download": 1024 + tick,
				"metadata": map[string]string{"host": "example.test", "destinationPort": "443", "network": "tcp"},
				"chains":   []string{"node-a"},
				"rule":     "MATCH",
			}},
		}
	})
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	stream(w, r, func(tick int64) any {
		return map[string]string{"type": "info", "payload": fmt.Sprintf("fixture log %d", tick)}
	})
}

func stream(w http.ResponseWriter, r *http.Request, message func(int64) any) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	encoder := json.NewEncoder(w)
	for tick := int64(1); ; tick++ {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := encoder.Encode(message(tick)); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
