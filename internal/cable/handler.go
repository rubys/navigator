package cable

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message types for the custom WebSocket protocol
type Message struct {
	Type   string          `json:"type"`
	Stream string          `json:"stream,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

// Connection represents a WebSocket client connection
type Connection struct {
	ws        *websocket.Conn
	streams   map[string]bool
	streamsMu sync.RWMutex
	send      chan []byte
	handler   *Handler
}

// Handler manages WebSocket connections and broadcasts
type Handler struct {
	connections   map[*Connection]bool
	connectionsMu sync.RWMutex
	streams       map[string]map[*Connection]bool // stream -> connections
	streamsMu     sync.RWMutex
	upgrader      websocket.Upgrader
	logger        *slog.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		connections: make(map[*Connection]bool),
		streams:     make(map[string]map[*Connection]bool),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Authentication handled by Navigator
			},
		},
		logger: logger,
	}
}

// ServeHTTP handles WebSocket upgrade requests at /cable
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", "error", err)
		return
	}

	conn := &Connection{
		ws:      ws,
		streams: make(map[string]bool),
		send:    make(chan []byte, 256),
		handler: h,
	}

	h.register(conn)
	defer h.unregister(conn)

	// Start write pump
	go conn.writePump()

	// Start read pump (blocks until connection closes)
	conn.readPump()
}

// HandleBroadcast handles HTTP POST broadcasts from Rails at /_broadcast
func (h *Handler) HandleBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if msg.Stream == "" {
		http.Error(w, "Stream required", http.StatusBadRequest)
		return
	}

	// Broadcast to all connections subscribed to this stream
	data, _ := json.Marshal(Message{
		Type:   "message",
		Stream: msg.Stream,
		Data:   msg.Data,
	})

	// Copy connections while holding lock to avoid race conditions
	h.streamsMu.RLock()
	streamConns := h.streams[msg.Stream]
	connections := make([]*Connection, 0, len(streamConns))
	for conn := range streamConns {
		connections = append(connections, conn)
	}
	h.streamsMu.RUnlock()

	count := 0
	for _, conn := range connections {
		select {
		case conn.send <- data:
			count++
		default:
			// Connection buffer full, skip
			h.logger.Warn("Dropped message", "stream", msg.Stream)
		}
	}

	h.logger.Debug("Broadcast sent", "stream", msg.Stream, "connections", count)
	w.WriteHeader(http.StatusOK)
}

// register adds a connection to the handler
func (h *Handler) register(conn *Connection) {
	h.connectionsMu.Lock()
	h.connections[conn] = true
	total := len(h.connections)
	h.connectionsMu.Unlock()
	h.logger.Debug("WebSocket connected", "total", total)
}

// unregister removes a connection and all its subscriptions
func (h *Handler) unregister(conn *Connection) {
	h.connectionsMu.Lock()
	delete(h.connections, conn)
	total := len(h.connections)
	h.connectionsMu.Unlock()

	conn.streamsMu.RLock()
	streams := make([]string, 0, len(conn.streams))
	for stream := range conn.streams {
		streams = append(streams, stream)
	}
	conn.streamsMu.RUnlock()

	for _, stream := range streams {
		h.unsubscribe(conn, stream)
	}

	close(conn.send)
	h.logger.Debug("WebSocket disconnected", "total", total)
}

// subscribe adds a connection to a stream
func (h *Handler) subscribe(conn *Connection, stream string) {
	h.streamsMu.Lock()
	if h.streams[stream] == nil {
		h.streams[stream] = make(map[*Connection]bool)
	}
	h.streams[stream][conn] = true
	h.streamsMu.Unlock()

	conn.streamsMu.Lock()
	conn.streams[stream] = true
	conn.streamsMu.Unlock()

	h.logger.Debug("Subscribed", "stream", stream)
}

// unsubscribe removes a connection from a stream
func (h *Handler) unsubscribe(conn *Connection, stream string) {
	h.streamsMu.Lock()
	if conns, ok := h.streams[stream]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.streams, stream)
		}
	}
	h.streamsMu.Unlock()

	conn.streamsMu.Lock()
	delete(conn.streams, stream)
	conn.streamsMu.Unlock()

	h.logger.Debug("Unsubscribed", "stream", stream)
}

// Shutdown gracefully closes all connections
func (h *Handler) Shutdown(ctx context.Context) error {
	h.connectionsMu.RLock()
	connections := make([]*Connection, 0, len(h.connections))
	for conn := range h.connections {
		connections = append(connections, conn)
	}
	h.connectionsMu.RUnlock()

	for _, conn := range connections {
		conn.ws.Close()
	}

	h.logger.Info("WebSocket handler shutdown complete")
	return nil
}

// readPump handles incoming messages from the WebSocket
func (conn *Connection) readPump() {
	defer conn.ws.Close()

	_ = conn.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.ws.SetPongHandler(func(string) error {
		_ = conn.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.ws.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "subscribe":
			if msg.Stream != "" {
				conn.handler.subscribe(conn, msg.Stream)
				// Send confirmation
				response, _ := json.Marshal(Message{
					Type:   "subscribed",
					Stream: msg.Stream,
				})
				conn.send <- response
			}

		case "unsubscribe":
			if msg.Stream != "" {
				conn.handler.unsubscribe(conn, msg.Stream)
			}

		case "pong":
			// Pong received, reset deadline
			_ = conn.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		}
	}
}

// writePump sends messages to the WebSocket
func (conn *Connection) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		conn.ws.Close()
	}()

	for {
		select {
		case message, ok := <-conn.send:
			_ = conn.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = conn.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.ws.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = conn.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			ping, _ := json.Marshal(Message{Type: "ping"})
			if err := conn.ws.WriteMessage(websocket.TextMessage, ping); err != nil {
				return
			}
		}
	}
}
