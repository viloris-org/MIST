package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

//go:embed build
var staticFiles embed.FS

// StatusProvider provides runtime metrics for the dashboard.
type StatusProvider interface {
	StatusJSON() ([]byte, error)
}

// ConfigProvider provides configuration data for the settings page.
// If a StatusProvider also implements ConfigProvider, the settings API is enabled.
type ConfigProvider interface {
	ConfigJSON() ([]byte, error)
}

// Dashboard serves an embedded web UI with a REST API.
type Dashboard struct {
	addr     string
	server   *http.Server
	provider StatusProvider

	passwordHash []byte
	tokens       map[string]time.Time // SHA-256(token) -> expiry
	tokensMu     sync.Mutex
	tokenDur     time.Duration
	tlsCertFile  string
	tlsKeyFile   string

	hub        *WSHub
	logBuffer  *LogBuffer
	logHook    *LogrusHook
	stopTicker   chan struct{}
	stopCleanup  chan struct{}
	stopOnce     sync.Once
	cleanupOnce  sync.Once
}

// New creates a new dashboard server.
func New(addr string, provider StatusProvider, opts ...Option) *Dashboard {
	hub := NewWSHub()
	logBuf := NewLogBuffer(hub)
	d := &Dashboard{
		addr:        addr,
		provider:    provider,
		tokenDur:    24 * time.Hour,
		tokens:      make(map[string]time.Time),
		hub:         hub,
		logBuffer:   logBuf,
		logHook:     NewLogrusHook(logBuf),
		stopTicker:  make(chan struct{}),
		stopCleanup: make(chan struct{}),
	}

	for _, o := range opts {
		o(d)
	}

	logrus.AddHook(d.logHook)
	return d
}

// Option configures a Dashboard.
type Option func(*Dashboard)

// WithPassword sets the dashboard login password.
func WithPassword(password string) Option {
	return func(d *Dashboard) {
		h := sha256.Sum256([]byte(password))
		d.passwordHash = h[:]
	}
}

// WithPasswordHash sets the password from a pre-computed SHA-256 hex hash.
func WithPasswordHash(hash string) Option {
	return func(d *Dashboard) {
		h, err := hex.DecodeString(hash)
		if err != nil {
			logrus.Errorf("Dashboard: invalid password hash: %v", err)
			return
		}
		d.passwordHash = h
	}
}

// WithTokenDuration sets the login session duration.
func WithTokenDuration(dur time.Duration) Option {
	return func(d *Dashboard) {
		d.tokenDur = dur
	}
}

// WithTLS sets the TLS certificate and key file paths.
func WithTLS(certFile, keyFile string) Option {
	return func(d *Dashboard) {
		d.tlsCertFile = certFile
		d.tlsKeyFile = keyFile
	}
}

// Start begins serving the dashboard.
func (d *Dashboard) Start() error {
	mux := http.NewServeMux()

	buildFS, err := fs.Sub(staticFiles, "build")
	if err != nil {
		return err
	}
	staticFS := http.FileServer(http.FS(buildFS))
	mux.Handle("/", staticFS)

	// Public endpoints (no auth).
	mux.HandleFunc("/api/login", d.handleLogin)
	mux.HandleFunc("/api/check", d.handleCheck)

	// Protected endpoints (require auth).
	mux.Handle("/api/status", d.authMiddleware(d.handleStatus))
	mux.Handle("/api/logout", d.authMiddleware(d.handleLogout))
	mux.Handle("/api/logs", d.authMiddleware(d.handleLogs))
	mux.Handle("/api/ws", d.authMiddleware(d.handleWS))
	mux.Handle("/api/config", d.authMiddleware(d.handleConfig))

	// Cleanup expired tokens periodically.
	go d.cleanupTokens()

	// Push status via WebSocket every 2 seconds.
	go d.statusTicker()

	d.server = &http.Server{
		Addr:    d.addr,
		Handler: mux,
	}

	go func() {
		proto := "http"
		if d.tlsCertFile != "" && d.tlsKeyFile != "" {
			proto = "https"
		}
		logrus.Infof("Dashboard listening on %s://%s", proto, d.addr)
		var serveErr error
		if d.tlsCertFile != "" && d.tlsKeyFile != "" {
			serveErr = d.server.ListenAndServeTLS(d.tlsCertFile, d.tlsKeyFile)
		} else {
			serveErr = d.server.ListenAndServe()
		}
		if serveErr != nil && serveErr != http.ErrServerClosed {
			logrus.Errorf("Dashboard: %v", serveErr)
		}
	}()

	return nil
}

// Stop shuts down the dashboard.
func (d *Dashboard) Stop() error {
	d.stopOnce.Do(func() {
		close(d.stopTicker)
	})
	d.cleanupOnce.Do(func() {
		close(d.stopCleanup)
	})
	// Mark hook inactive so it stops producing entries for a closed hub.
	d.logHook.stopped.Store(true)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return d.server.Shutdown(ctx)
}

// --- Auth handlers ---

type loginRequest struct {
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

func (d *Dashboard) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	if d.passwordHash == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "dashboard password not configured"})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	// Constant-time password comparison.
	given := sha256.Sum256([]byte(req.Password))
	if subtle.ConstantTimeCompare(d.passwordHash, given[:]) != 1 {
		// Add a small delay to throttle brute force.
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid password"})
		logrus.Warn("Dashboard: failed login attempt")
		return
	}

	// Generate token: 32 random bytes, hex-encoded.
	raw := make([]byte, 32)
	rand.Read(raw)
	token := hex.EncodeToString(raw)

	// Store SHA-256(token) for validation.
	h := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(h[:])
	expiry := time.Now().Add(d.tokenDur)

	d.tokensMu.Lock()
	d.tokens[tokenHash] = expiry
	d.tokensMu.Unlock()

	logrus.Info("Dashboard: login successful")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginResponse{
		Token:     token,
		ExpiresAt: expiry.Format(time.RFC3339),
	})
}

func (d *Dashboard) handleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	token := extractToken(r)
	if token != "" {
		h := sha256.Sum256([]byte(token))
		d.tokensMu.Lock()
		delete(d.tokens, hex.EncodeToString(h[:]))
		d.tokensMu.Unlock()
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "logged out"})
}

func (d *Dashboard) handleCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	authenticated := d.validateToken(extractToken(r))
	body := map[string]bool{"authenticated": authenticated}
	if d.passwordHash == nil {
		body["no_password_set"] = true
		body["authenticated"] = true
	}
	json.NewEncoder(w).Encode(body)
}

// --- Auth middleware ---

func (d *Dashboard) authMiddleware(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")

		// If no password is set, allow all requests.
		if d.passwordHash == nil {
			next(w, r)
			return
		}

		token := extractToken(r)
		if !d.validateToken(token) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		next(w, r)
	})
}

// --- Token helpers ---

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth != "" {
		if token, ok := strings.CutPrefix(auth, "Bearer "); ok {
			return token
		}
	}
	return r.URL.Query().Get("token")
}

func (d *Dashboard) validateToken(token string) bool {
	if token == "" {
		return false
	}
	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	d.tokensMu.Lock()
	expiry, ok := d.tokens[key]
	if ok {
		if time.Now().After(expiry) {
			delete(d.tokens, key)
			ok = false
		}
	}
	d.tokensMu.Unlock()
	return ok
}

func (d *Dashboard) cleanupTokens() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			d.tokensMu.Lock()
			now := time.Now()
			for k, exp := range d.tokens {
				if now.After(exp) {
					delete(d.tokens, k)
				}
			}
			d.tokensMu.Unlock()
		case <-d.stopCleanup:
			return
		}
	}
}

// --- API handlers ---

func (d *Dashboard) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

	// Also push via WebSocket.
	d.hub.BroadcastStatus(data)

	w.Write(data)
}

func (d *Dashboard) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	cp, ok := d.provider.(ConfigProvider)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "config not available"})
		return
	}

	if r.Method == http.MethodPut {
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{"error": "config updates not yet supported"})
		return
	}

	data, err := cp.ConfigJSON()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Write(data)
}

func (d *Dashboard) handleWS(w http.ResponseWriter, r *http.Request) {
	d.hub.ServeHTTP(w, r)
}

func (d *Dashboard) handleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	entries := d.logBuffer.Snapshot()
	if entries == nil {
		entries = []LogEntry{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	})
}

func (d *Dashboard) statusTicker() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if d.provider != nil {
				if data, err := d.provider.StatusJSON(); err == nil {
					d.hub.BroadcastStatus(data)
				}
			}
		case <-d.stopTicker:
			return
		}
	}
}
