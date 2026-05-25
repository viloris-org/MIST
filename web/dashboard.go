package web

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

//go:embed static/*
var staticFiles embed.FS

// StatusProvider provides runtime metrics for the dashboard.
type StatusProvider interface {
	StatusJSON() ([]byte, error)
}

// Dashboard serves an embedded web UI with a REST API.
type Dashboard struct {
	addr     string
	server   *http.Server
	provider StatusProvider
	mu       sync.Mutex
}

// New creates a new dashboard server.
func New(addr string, provider StatusProvider) *Dashboard {
	return &Dashboard{
		addr:     addr,
		provider: provider,
	}
}

// Start begins serving the dashboard.
func (d *Dashboard) Start() error {
	mux := http.NewServeMux()

	// Serve static files.
	staticFS := http.FileServer(http.FS(staticFiles))
	mux.Handle("/static/", staticFS)
	mux.Handle("/", staticFS)

	// API endpoints.
	mux.HandleFunc("/api/status", d.handleStatus)

	d.server = &http.Server{
		Addr:    d.addr,
		Handler: mux,
	}

	go func() {
		logrus.Infof("Dashboard listening on http://%s", d.addr)
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("Dashboard: %v", err)
		}
	}()

	return nil
}

// Stop shuts down the dashboard.
func (d *Dashboard) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return d.server.Shutdown(ctx)
}

func (d *Dashboard) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if d.provider == nil {
		w.Write([]byte(`{"error":"no status provider"}`))
		return
	}

	data, err := d.provider.StatusJSON()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Write(data)
}
