package web

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Message types sent from server to client.
const (
	MsgTypeStatus  = "status"
	MsgTypeTraffic = "traffic"
	MsgTypeLog     = "log"
)

type wsMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Time    int64       `json:"time"`
}

// WSHub manages all WebSocket connections and broadcasts.
type WSHub struct {
	mu          sync.RWMutex
	clients     map[*wsClient]struct{}
	broadcast   chan wsMessage
	register    chan *wsClient
	unregister  chan *wsClient
	lastStatus  []byte
	lastTraffic []byte
	statusMu    sync.Mutex

	// Throttle per message type.
	lastSent map[string]time.Time
}

type wsClient struct {
	conn     *websocket.Conn
	send     chan []byte
	hub      *WSHub
	die      chan struct{}
}

// NewWSHub creates a new WebSocket hub and starts its run loop.
func NewWSHub() *WSHub {
	h := &WSHub{
		clients:    make(map[*wsClient]struct{}),
		broadcast:  make(chan wsMessage, 64),
		register:   make(chan *wsClient),
		unregister: make(chan *wsClient),
		lastSent:   make(map[string]time.Time),
	}
	go h.run()
	return h
}

func (h *WSHub) run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()

			// Send last known status on connect.
			h.statusMu.Lock()
			if h.lastStatus != nil {
				select {
				case c.send <- h.lastStatus:
				default:
				}
			}
			if h.lastTraffic != nil {
				select {
				case c.send <- h.lastTraffic:
				default:
				}
			}
			h.statusMu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			// Marshal once.
			data, err := json.Marshal(msg)
			if err != nil {
				h.mu.RUnlock()
				continue
			}

			// Throttle status/traffic to 250ms.
			if msg.Type == MsgTypeStatus || msg.Type == MsgTypeTraffic {
				now := time.Now()
				if last, ok := h.lastSent[msg.Type]; ok && now.Sub(last) < 250*time.Millisecond {
					h.mu.RUnlock()
					continue
				}
				h.lastSent[msg.Type] = now
			}

			// Cache status and traffic for new clients.
			if msg.Type == MsgTypeStatus {
				h.statusMu.Lock()
				h.lastStatus = data
				h.statusMu.Unlock()
			} else if msg.Type == MsgTypeTraffic {
				h.statusMu.Lock()
				h.lastTraffic = data
				h.statusMu.Unlock()
			}

			for c := range h.clients {
				select {
				case c.send <- data:
				default:
					// Client too slow — disconnect.
					go c.close()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *WSHub) Broadcast(msg wsMessage) {
	msg.Time = time.Now().UnixMilli()
	select {
	case h.broadcast <- msg:
	default:
		// Drop if broadcast channel is full.
	}
}

// BroadcastStatus sends a status update. Throttled by the hub.
func (h *WSHub) BroadcastStatus(metrics []byte) {
	// Try to parse as JSON to send as structured payload.
	var payload interface{}
	if err := json.Unmarshal(metrics, &payload); err != nil {
		payload = json.RawMessage(metrics)
	}
	h.Broadcast(wsMessage{Type: MsgTypeStatus, Payload: payload})
}

// BroadcastTraffic sends traffic data.
func (h *WSHub) BroadcastTraffic(data interface{}) {
	h.Broadcast(wsMessage{Type: MsgTypeTraffic, Payload: data})
}

// BroadcastLog sends a log entry.
func (h *WSHub) BroadcastLog(entry LogEntry) {
	h.Broadcast(wsMessage{Type: MsgTypeLog, Payload: entry})
}

// ServeHTTP handles WebSocket upgrade requests.
func (h *WSHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Debugf("WS upgrade: %v", err)
		return
	}

	c := &wsClient{
		conn: conn,
		send: make(chan []byte, 32),
		hub:  h,
		die:  make(chan struct{}),
	}

	h.register <- c

	go c.writePump()
	go c.readPump()
}

func (c *wsClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(4096)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *wsClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.die:
			return
		}
	}
}

func (c *wsClient) close() {
	select {
	case <-c.die:
	default:
		close(c.die)
	}
}
