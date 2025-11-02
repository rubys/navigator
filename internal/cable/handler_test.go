package cable

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewHandler(logger)

	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}

	if handler.connections == nil {
		t.Error("connections map not initialized")
	}

	if handler.streams == nil {
		t.Error("streams map not initialized")
	}

	if handler.logger == nil {
		t.Error("logger not set")
	}
}

func TestHandleBroadcast_ValidRequest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewHandler(logger)

	// Create a broadcast request
	msg := Message{
		Stream: "test-stream",
		Data:   json.RawMessage(`"test data"`),
	}
	body, _ := json.Marshal(msg)

	req := httptest.NewRequest("POST", "/_broadcast", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleBroadcast(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleBroadcast_InvalidMethod(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewHandler(logger)

	req := httptest.NewRequest("GET", "/_broadcast", nil)
	w := httptest.NewRecorder()

	handler.HandleBroadcast(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleBroadcast_InvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewHandler(logger)

	req := httptest.NewRequest("POST", "/_broadcast", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleBroadcast(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleBroadcast_MissingStream(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewHandler(logger)

	msg := Message{
		Data: json.RawMessage(`"test data"`),
		// Stream is empty
	}
	body, _ := json.Marshal(msg)

	req := httptest.NewRequest("POST", "/_broadcast", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleBroadcast(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestWebSocketUpgrade(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewHandler(logger)

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect as WebSocket client
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Verify we can send a message
	subscribe := Message{
		Type:   "subscribe",
		Stream: "test-stream",
	}
	if err := ws.WriteJSON(subscribe); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Read response
	var response Message
	if err := ws.ReadJSON(&response); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != "subscribed" {
		t.Errorf("Expected type 'subscribed', got '%s'", response.Type)
	}

	if response.Stream != "test-stream" {
		t.Errorf("Expected stream 'test-stream', got '%s'", response.Stream)
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewHandler(logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Subscribe
	subscribe := Message{Type: "subscribe", Stream: "test-stream"}
	if err := ws.WriteJSON(subscribe); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Read confirmation
	var response Message
	if err := ws.ReadJSON(&response); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != "subscribed" {
		t.Errorf("Expected 'subscribed', got '%s'", response.Type)
	}

	// Verify subscription was registered
	handler.streamsMu.RLock()
	subscribers := len(handler.streams["test-stream"])
	handler.streamsMu.RUnlock()

	if subscribers != 1 {
		t.Errorf("Expected 1 subscriber, got %d", subscribers)
	}

	// Unsubscribe
	unsubscribe := Message{Type: "unsubscribe", Stream: "test-stream"}
	if err := ws.WriteJSON(unsubscribe); err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	// Give it a moment to process
	time.Sleep(50 * time.Millisecond)

	// Verify subscription was removed
	handler.streamsMu.RLock()
	subscribers = len(handler.streams["test-stream"])
	handler.streamsMu.RUnlock()

	if subscribers != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", subscribers)
	}
}

func TestBroadcastToSubscribers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewHandler(logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect two WebSocket clients
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}
	defer ws2.Close()

	// Both subscribe to same stream
	subscribe := Message{Type: "subscribe", Stream: "broadcast-test"}

	if err := ws1.WriteJSON(subscribe); err != nil {
		t.Fatalf("Client 1 failed to subscribe: %v", err)
	}
	if err := ws2.WriteJSON(subscribe); err != nil {
		t.Fatalf("Client 2 failed to subscribe: %v", err)
	}

	// Read confirmations
	var response Message
	ws1.ReadJSON(&response)
	ws2.ReadJSON(&response)

	// Send broadcast via HTTP
	broadcastMsg := Message{
		Stream: "broadcast-test",
		Data:   json.RawMessage(`{"message":"hello"}`),
	}
	body, _ := json.Marshal(broadcastMsg)

	req := httptest.NewRequest("POST", "/_broadcast", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleBroadcast(w, req)

	// Both clients should receive the broadcast
	var msg1, msg2 Message

	ws1.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := ws1.ReadJSON(&msg1); err != nil {
		t.Fatalf("Client 1 failed to receive broadcast: %v", err)
	}

	ws2.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := ws2.ReadJSON(&msg2); err != nil {
		t.Fatalf("Client 2 failed to receive broadcast: %v", err)
	}

	// Verify both received the message
	if msg1.Type != "message" || msg1.Stream != "broadcast-test" {
		t.Errorf("Client 1 received unexpected message: %+v", msg1)
	}

	if msg2.Type != "message" || msg2.Stream != "broadcast-test" {
		t.Errorf("Client 2 received unexpected message: %+v", msg2)
	}
}

func TestPingPong(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewHandler(logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Send pong message
	pong := Message{Type: "pong"}
	if err := ws.WriteJSON(pong); err != nil {
		t.Fatalf("Failed to send pong: %v", err)
	}

	// Wait briefly to ensure no error occurs
	time.Sleep(50 * time.Millisecond)

	// Connection should still be alive - send subscribe to verify
	subscribe := Message{Type: "subscribe", Stream: "test"}
	if err := ws.WriteJSON(subscribe); err != nil {
		t.Fatalf("Connection died after pong: %v", err)
	}

	var response Message
	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := ws.ReadJSON(&response); err != nil {
		t.Fatalf("Failed to read after pong: %v", err)
	}

	if response.Type != "subscribed" {
		t.Errorf("Expected subscribed response, got %s", response.Type)
	}
}

func TestShutdown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewHandler(logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Subscribe to verify connection is working
	subscribe := Message{Type: "subscribe", Stream: "test"}
	if err := ws.WriteJSON(subscribe); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	var response Message
	ws.ReadJSON(&response)

	// Shutdown the handler
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := handler.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify connection was closed
	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, _, err = ws.ReadMessage()
	if err == nil {
		t.Error("Expected connection to be closed after shutdown")
	}
}

func TestMultipleStreams(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewHandler(logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Subscribe to multiple streams
	streams := []string{"stream1", "stream2", "stream3"}
	for _, stream := range streams {
		subscribe := Message{Type: "subscribe", Stream: stream}
		if err := ws.WriteJSON(subscribe); err != nil {
			t.Fatalf("Failed to subscribe to %s: %v", stream, err)
		}

		var response Message
		ws.ReadJSON(&response)

		if response.Stream != stream {
			t.Errorf("Expected stream %s, got %s", stream, response.Stream)
		}
	}

	// Broadcast to one stream
	broadcastMsg := Message{
		Stream: "stream2",
		Data:   json.RawMessage(`{"test":true}`),
	}
	body, _ := json.Marshal(broadcastMsg)

	req := httptest.NewRequest("POST", "/_broadcast", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleBroadcast(w, req)

	// Should receive message on stream2
	var msg Message
	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := ws.ReadJSON(&msg); err != nil {
		t.Fatalf("Failed to receive broadcast: %v", err)
	}

	if msg.Stream != "stream2" {
		t.Errorf("Expected stream2, got %s", msg.Stream)
	}

	// Verify we're still subscribed to all streams
	handler.connectionsMu.RLock()
	var conn *Connection
	for c := range handler.connections {
		conn = c
		break
	}
	handler.connectionsMu.RUnlock()

	if conn == nil {
		t.Fatal("No connection found")
	}

	conn.streamsMu.RLock()
	subscribedCount := len(conn.streams)
	conn.streamsMu.RUnlock()

	if subscribedCount != 3 {
		t.Errorf("Expected 3 subscribed streams, got %d", subscribedCount)
	}
}
