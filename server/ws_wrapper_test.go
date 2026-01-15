package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestWsTransportImplementsTransport verifies wsTransport implements Transport interface
func TestWsTransportImplementsTransport(t *testing.T) {
	// Compile-time check that wsTransport implements Transport
	var _ Transport = (*wsTransport)(nil)
}

// setupWebSocketPair creates a client-server WebSocket pair for testing
func setupWebSocketPair(t *testing.T) (*wsTransport, *websocket.Conn, func()) {
	t.Helper()

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	serverConnCh := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Upgrade error: %v", err)
			return
		}
		serverConnCh <- conn
	}))

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	select {
	case serverConn := <-serverConnCh:
		transport := &wsTransport{serverConn}
		return transport, clientConn, func() {
			clientConn.Close()
			serverConn.Close()
			server.Close()
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for server connection")
		return nil, nil, nil
	}
}

func TestWsTransportWrite(t *testing.T) {
	transport, clientConn, cleanup := setupWebSocketPair(t)
	defer cleanup()

	// Write from server transport
	testData := []byte("hello websocket")
	n, err := transport.Write(testData)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write() returned %d, expected %d", n, len(testData))
	}

	// Read from client
	msgType, msg, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("Client ReadMessage() error: %v", err)
	}
	if msgType != websocket.TextMessage {
		t.Errorf("Message type = %d, expected TextMessage (%d)", msgType, websocket.TextMessage)
	}
	if !bytes.Equal(msg, testData) {
		t.Errorf("Received message = %v, expected %v", msg, testData)
	}
}

func TestWsTransportRead(t *testing.T) {
	transport, clientConn, cleanup := setupWebSocketPair(t)
	defer cleanup()

	// Write from client
	testData := []byte("hello from client")
	err := clientConn.WriteMessage(websocket.TextMessage, testData)
	if err != nil {
		t.Fatalf("Client WriteMessage() error: %v", err)
	}

	// Read from server transport
	buf := make([]byte, 1024)
	n, err := transport.Read(buf)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if !bytes.Equal(buf[:n], testData) {
		t.Errorf("Read data = %v, expected %v", buf[:n], testData)
	}
}

func TestWsTransportReadSkipsBinaryMessages(t *testing.T) {
	transport, clientConn, cleanup := setupWebSocketPair(t)
	defer cleanup()

	// Write binary message first (should be skipped)
	err := clientConn.WriteMessage(websocket.BinaryMessage, []byte("binary data"))
	if err != nil {
		t.Fatalf("Client WriteMessage(binary) error: %v", err)
	}

	// Write text message
	textData := []byte("text message")
	err = clientConn.WriteMessage(websocket.TextMessage, textData)
	if err != nil {
		t.Fatalf("Client WriteMessage(text) error: %v", err)
	}

	// Read should return the text message
	buf := make([]byte, 1024)
	n, err := transport.Read(buf)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if !bytes.Equal(buf[:n], textData) {
		t.Errorf("Read data = %v, expected %v (binary should be skipped)", buf[:n], textData)
	}
}

func TestWsTransportClose(t *testing.T) {
	transport, _, cleanup := setupWebSocketPair(t)
	defer cleanup()

	err := transport.Close()
	if err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestWsTransportRemoteAddr(t *testing.T) {
	transport, _, cleanup := setupWebSocketPair(t)
	defer cleanup()

	addr := transport.RemoteAddr()
	if addr == "" {
		t.Error("RemoteAddr() returned empty string")
	}
	// Should be something like "127.0.0.1:xxxxx"
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Errorf("RemoteAddr() = %s, expected to start with 127.0.0.1:", addr)
	}
}

func TestWsTransportReadBufferOverflow(t *testing.T) {
	transport, clientConn, cleanup := setupWebSocketPair(t)
	defer cleanup()

	// Write data larger than read buffer
	largeData := make([]byte, 2000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	err := clientConn.WriteMessage(websocket.TextMessage, largeData)
	if err != nil {
		t.Fatalf("Client WriteMessage() error: %v", err)
	}

	// Read with small buffer - should fail
	smallBuf := make([]byte, 100)
	_, err = transport.Read(smallBuf)
	if err == nil {
		t.Error("Read() should fail when message exceeds buffer size")
	}
}

func TestWsTransportMultipleWriteRead(t *testing.T) {
	transport, clientConn, cleanup := setupWebSocketPair(t)
	defer cleanup()

	messages := []string{"message1", "message2", "message3"}

	// Write multiple messages
	for _, msg := range messages {
		_, err := transport.Write([]byte(msg))
		if err != nil {
			t.Fatalf("Write() error: %v", err)
		}
	}

	// Read all messages from client
	for _, expected := range messages {
		_, msg, err := clientConn.ReadMessage()
		if err != nil {
			t.Fatalf("Client ReadMessage() error: %v", err)
		}
		if string(msg) != expected {
			t.Errorf("Received %s, expected %s", msg, expected)
		}
	}
}

func TestWsTransportReadAfterClose(t *testing.T) {
	transport, clientConn, cleanup := setupWebSocketPair(t)
	defer cleanup()

	// Close client connection
	clientConn.Close()

	// Read should return error
	buf := make([]byte, 100)
	_, err := transport.Read(buf)
	if err == nil {
		t.Error("Read() should fail after connection close")
	}
}

// Benchmark tests
func BenchmarkWsTransportWrite(b *testing.B) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	serverConnCh := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		serverConnCh <- conn
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	defer clientConn.Close()

	serverConn := <-serverConnCh
	defer serverConn.Close()

	transport := &wsTransport{serverConn}

	// Start a goroutine to read messages
	go func() {
		for {
			_, _, err := clientConn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	data := []byte("benchmark test data")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		transport.Write(data)
	}
}

// Test that wsTransport implements io.ReadWriter
func TestWsTransportAsReadWriter(t *testing.T) {
	transport, clientConn, cleanup := setupWebSocketPair(t)
	defer cleanup()

	// Use io.Copy pattern
	testData := []byte("io.ReadWriter test")
	_, err := transport.Write(testData)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}

	_, msg, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("Client read error: %v", err)
	}

	if !bytes.Equal(msg, testData) {
		t.Errorf("Data mismatch: got %v, want %v", msg, testData)
	}

	// Verify interface compliance
	var rw io.ReadWriter = transport
	_ = rw
}
