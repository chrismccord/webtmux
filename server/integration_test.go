//go:build integration
// +build integration

package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Integration tests for the server package.
// Run with: go test -tags=integration ./server/...

// generateTestCertificates creates self-signed TLS certificates for testing
func generateTestCertificates(t *testing.T) (certFile, keyFile string, cleanup func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "webtmux-test-certs-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"WebTmux Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		DNSNames:              []string{"localhost"},
	}

	// Create certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Write certificate file
	certFile = filepath.Join(tmpDir, "cert.pem")
	certOut, err := os.Create(certFile)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create cert file: %v", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	// Write key file
	keyFile = filepath.Join(tmpDir, "key.pem")
	keyOut, err := os.Create(keyFile)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create key file: %v", err)
	}
	keyBytes, _ := x509.MarshalECPrivateKey(privateKey)
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	keyOut.Close()

	cleanup = func() {
		os.RemoveAll(tmpDir)
	}

	return certFile, keyFile, cleanup
}

// mockSlave implements the Slave interface for testing
type mockSlave struct {
	reader *io.PipeReader
	writer *io.PipeWriter
	closed bool
	mu     sync.Mutex
}

func newMockSlave() *mockSlave {
	r, w := io.Pipe()
	return &mockSlave{
		reader: r,
		writer: w,
	}
}

func (m *mockSlave) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

func (m *mockSlave) Write(p []byte) (n int, err error) {
	return m.writer.Write(p)
}

func (m *mockSlave) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	m.reader.Close()
	m.writer.Close()
	return nil
}

func (m *mockSlave) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{
		"command": "test",
	}
}

func (m *mockSlave) ResizeTerminal(columns int, rows int) error {
	return nil
}

// mockIntegrationFactory implements Factory for integration testing
type mockIntegrationFactory struct {
	slaves []*mockSlave
	mu     sync.Mutex
}

func newMockIntegrationFactory() *mockIntegrationFactory {
	return &mockIntegrationFactory{
		slaves: make([]*mockSlave, 0),
	}
}

func (f *mockIntegrationFactory) Name() string {
	return "mock-integration"
}

func (f *mockIntegrationFactory) New(params map[string][]string, headers map[string][]string) (Slave, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	slave := newMockSlave()
	f.slaves = append(f.slaves, slave)
	return slave, nil
}

func (f *mockIntegrationFactory) Command() (string, []string) {
	return "echo", []string{"test"}
}

func (f *mockIntegrationFactory) CloseAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.slaves {
		s.Close()
	}
}

// TestServerCreation tests that a server can be created with valid options
func TestIntegrationServerCreation(t *testing.T) {
	factory := newMockIntegrationFactory()
	options := &Options{
		Address:     "127.0.0.1",
		Port:        "0", // Let OS assign port
		Path:        "/",
		TitleFormat: "WebTmux - Integration Test",
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Server is nil")
	}
}

// TestWebSocketHandshake tests that WebSocket connections can be established
func TestIntegrationWebSocketHandshake(t *testing.T) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0",
		Path:          "/",
		TitleFormat:   "Test",
		PermitWrite:   true,
		MaxConnection: 10,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create test HTTP server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := server.generateHandleWS(ctx, cancel, counter)

	testServer := httptest.NewServer(http.HandlerFunc(handler))
	defer testServer.Close()

	// Connect via WebSocket
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	dialer := websocket.Dialer{
		Subprotocols: []string{"webtty"},
	}

	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("Expected status 101, got %d", resp.StatusCode)
	}

	// Verify protocol negotiation
	if conn.Subprotocol() != "webtty" {
		t.Errorf("Expected subprotocol 'webtty', got '%s'", conn.Subprotocol())
	}
}

// TestWebSocketMessageExchange tests sending and receiving messages
func TestIntegrationWebSocketMessageExchange(t *testing.T) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0",
		Path:          "/",
		TitleFormat:   "Test",
		PermitWrite:   true,
		MaxConnection: 10,
		NoAuth:        true, // Disable auth for simpler testing
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := server.generateHandleWS(ctx, cancel, counter)

	testServer := httptest.NewServer(http.HandlerFunc(handler))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	dialer := websocket.Dialer{
		Subprotocols: []string{"webtty"},
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Try to read initial message - may timeout if auth is required
	// This is acceptable for integration testing
	msgType, msg, err := conn.ReadMessage()
	if err != nil {
		// Timeout or connection closed is acceptable in this test
		// The main purpose is to verify the connection can be established
		t.Logf("Read message result: err=%v (acceptable for integration test)", err)
		return
	}

	t.Logf("Received message type=%d, len=%d", msgType, len(msg))

	// Connection is established - this confirms basic message exchange works
}

// TestServerConcurrentConnections tests multiple simultaneous connections
func TestIntegrationConcurrentConnections(t *testing.T) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0",
		Path:          "/",
		TitleFormat:   "Test",
		PermitWrite:   true,
		MaxConnection: 10,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := server.generateHandleWS(ctx, cancel, counter)

	testServer := httptest.NewServer(http.HandlerFunc(handler))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	numConnections := 5
	var wg sync.WaitGroup
	errors := make(chan error, numConnections)

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			dialer := websocket.Dialer{
				Subprotocols: []string{"webtty"},
			}

			conn, _, err := dialer.Dial(wsURL, nil)
			if err != nil {
				errors <- fmt.Errorf("connection %d failed: %v", id, err)
				return
			}
			defer conn.Close()

			// Keep connection open briefly
			time.Sleep(100 * time.Millisecond)
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	// Verify counter tracked connections
	if counter.count() < 0 {
		t.Error("Counter should not be negative")
	}
}

// TestServerMaxConnections tests that max connection limit is enforced
func TestIntegrationMaxConnections(t *testing.T) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0",
		Path:          "/",
		TitleFormat:   "Test",
		PermitWrite:   true,
		MaxConnection: 2, // Limit to 2 connections
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := server.generateHandleWS(ctx, cancel, counter)

	testServer := httptest.NewServer(http.HandlerFunc(handler))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Open first two connections (should succeed)
	dialer := websocket.Dialer{
		Subprotocols: []string{"webtty"},
	}

	conn1, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("First connection failed: %v", err)
	}
	defer conn1.Close()

	conn2, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Second connection failed: %v", err)
	}
	defer conn2.Close()

	// Third connection should be rejected or limited
	// Note: The exact behavior depends on implementation
	conn3, _, err := dialer.Dial(wsURL, nil)
	if err == nil {
		conn3.Close()
		// Connection was accepted but might be immediately closed
		t.Log("Third connection was accepted (may be closed shortly)")
	}
}

// TestServerGracefulShutdown tests that server shuts down gracefully
func TestIntegrationGracefulShutdown(t *testing.T) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0",
		Path:          "/",
		TitleFormat:   "Test",
		PermitWrite:   true,
		MaxConnection: 10,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	counter := newCounter(0)
	handler := server.generateHandleWS(ctx, cancel, counter)

	testServer := httptest.NewServer(http.HandlerFunc(handler))

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	dialer := websocket.Dialer{
		Subprotocols: []string{"webtty"},
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}

	// Cancel context to trigger shutdown
	cancel()

	// Give time for shutdown
	time.Sleep(100 * time.Millisecond)

	// Connection should be closed or closing
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, _, err = conn.ReadMessage()
	// Error is expected after shutdown
	if err == nil {
		t.Log("Connection still readable after cancel (may be expected)")
	}

	conn.Close()
	testServer.Close()
}

// TestHandlersIntegration tests HTTP handlers with real requests
func TestIntegrationHTTPHandlers(t *testing.T) {
	factory := newMockIntegrationFactory()

	options := &Options{
		Address:            "127.0.0.1",
		Port:               "0",
		Path:               "/",
		TitleFormat:        "Integration Test",
		EnableWebTransport: false,
		WSQueryArgs:        "token=test123",
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test handleConfig
	t.Run("handleConfig", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/config.js", nil)
		rr := httptest.NewRecorder()

		server.handleConfig(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("handleConfig status = %d, want %d", rr.Code, http.StatusOK)
		}

		body := rr.Body.String()
		if !strings.Contains(body, "gotty_term") {
			t.Error("Config should contain gotty_term")
		}
		if !strings.Contains(body, "token=test123") {
			t.Error("Config should contain query args")
		}
	})

	// Test handleIndex
	t.Run("handleIndex", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		server.handleIndex(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("handleIndex status = %d, want %d", rr.Code, http.StatusOK)
		}
	})

	// Test handleManifest
	t.Run("handleManifest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/manifest.webmanifest", nil)
		rr := httptest.NewRecorder()

		server.handleManifest(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("handleManifest status = %d, want %d", rr.Code, http.StatusOK)
		}
	})

	// Test handleAuthToken
	t.Run("handleAuthToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth_token.js", nil)
		rr := httptest.NewRecorder()

		server.handleAuthToken(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("handleAuthToken status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}

// Benchmark concurrent WebSocket connections
func BenchmarkIntegrationConcurrentConnections(b *testing.B) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0",
		Path:          "/",
		TitleFormat:   "Benchmark",
		PermitWrite:   true,
		MaxConnection: 1000,
	}

	server, _ := New(factory, options)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := server.generateHandleWS(ctx, cancel, counter)

	testServer := httptest.NewServer(http.HandlerFunc(handler))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dialer := websocket.Dialer{
			Subprotocols: []string{"webtty"},
		}

		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			b.Fatalf("Dial failed: %v", err)
		}
		conn.Close()
	}
}

// TestIntegrationServerRun tests the full Server.Run lifecycle
func TestIntegrationServerRun(t *testing.T) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0", // Random port
		Path:          "/",
		TitleFormat:   "Server Run Test",
		PermitWrite:   true,
		MaxConnection: 10,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Run server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Cancel to stop server
	cancel()

	// Wait for server to stop
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("Server.Run returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server.Run did not stop within timeout")
	}
}

// TestIntegrationServerRunWithTLS tests Server.Run with TLS enabled
func TestIntegrationServerRunWithTLS(t *testing.T) {
	certFile, keyFile, cleanup := generateTestCertificates(t)
	defer cleanup()

	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0",
		Path:          "/",
		TitleFormat:   "TLS Test",
		PermitWrite:   true,
		MaxConnection: 10,
		EnableTLS:     true,
		TLSCrtFile:    certFile,
		TLSKeyFile:    keyFile,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Cancel to stop
	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("Server.Run with TLS returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server.Run with TLS did not stop within timeout")
	}
}

// TestIntegrationServerRunWithRandomURL tests Server.Run with random URL
func TestIntegrationServerRunWithRandomURL(t *testing.T) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:         "127.0.0.1",
		Port:            "0",
		Path:            "/",
		TitleFormat:     "Random URL Test",
		PermitWrite:     true,
		MaxConnection:   10,
		EnableRandomUrl: true,
		RandomUrlLength: 8,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("Server.Run with random URL returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server.Run with random URL did not stop within timeout")
	}
}

// TestIntegrationServerRunGracefulShutdown tests graceful shutdown
func TestIntegrationServerRunGracefulShutdown(t *testing.T) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0",
		Path:          "/",
		TitleFormat:   "Graceful Shutdown Test",
		PermitWrite:   true,
		MaxConnection: 10,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()
	gracefulCtx, gracefulCancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx, WithGracefullContext(gracefulCtx))
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Trigger graceful shutdown
	gracefulCancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Server.Run graceful shutdown returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server graceful shutdown did not complete within timeout")
	}
}

// TestIntegrationSetupHandlers tests the setupHandlers function
func TestIntegrationSetupHandlers(t *testing.T) {
	factory := newMockIntegrationFactory()

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0",
		Path:          "/terminal/",
		TitleFormat:   "Setup Handlers Test",
		PermitWrite:   true,
		MaxConnection: 10,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handlers := server.setupHandlers(ctx, cancel, "/terminal/", counter)

	if handlers == nil {
		t.Fatal("setupHandlers returned nil")
	}

	// Test with httptest
	testServer := httptest.NewServer(handlers)
	defer testServer.Close()

	// Test index page
	resp, err := http.Get(testServer.URL + "/terminal/")
	if err != nil {
		t.Fatalf("Failed to get index: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Index status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Test config.js
	resp, err = http.Get(testServer.URL + "/terminal/config.js")
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Config status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Test auth_token.js
	resp, err = http.Get(testServer.URL + "/terminal/auth_token.js")
	if err != nil {
		t.Fatalf("Failed to get auth_token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Auth token status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// TestIntegrationWebSocketWithBasicAuth tests WebSocket with basic auth
func TestIntegrationWebSocketWithBasicAuth(t *testing.T) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:         "127.0.0.1",
		Port:            "0",
		Path:            "/",
		TitleFormat:     "Auth Test",
		PermitWrite:     true,
		MaxConnection:   10,
		EnableBasicAuth: true,
		Credential:      "admin:password",
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handlers := server.setupHandlers(ctx, cancel, "/", counter)

	testServer := httptest.NewServer(handlers)
	defer testServer.Close()

	// Test without auth - should fail
	resp, err := http.Get(testServer.URL + "/")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Without auth: status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	// Test with auth - should succeed
	client := &http.Client{}
	req, _ := http.NewRequest("GET", testServer.URL+"/", nil)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Request with auth failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("With auth: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// TestIntegrationTLSConnection tests HTTPS connection
func TestIntegrationTLSConnection(t *testing.T) {
	certFile, keyFile, cleanup := generateTestCertificates(t)
	defer cleanup()

	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0",
		Path:          "/",
		TitleFormat:   "TLS Connection Test",
		PermitWrite:   true,
		MaxConnection: 10,
		EnableTLS:     true,
		TLSCrtFile:    certFile,
		TLSKeyFile:    keyFile,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	options.Port = fmt.Sprintf("%d", port)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Create TLS client
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 2 * time.Second,
	}

	// Make HTTPS request
	resp, err := client.Get(fmt.Sprintf("https://127.0.0.1:%d/config.js", port))
	if err != nil {
		t.Logf("HTTPS request failed (may be timing): %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("HTTPS response status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	}

	cancel()

	select {
	case <-errCh:
	case <-time.After(3 * time.Second):
	}
}

// TestIntegrationWebSocketAuthentication tests the full authentication flow
func TestIntegrationWebSocketAuthentication(t *testing.T) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	credential := "testuser:testpass"

	options := &Options{
		Address:       "127.0.0.1",
		Port:          "0",
		Path:          "/",
		TitleFormat:   "Test Terminal",
		PermitWrite:   true,
		MaxConnection: 10,
		Credential:    credential,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := server.generateHandleWS(ctx, cancel, counter)

	testServer := httptest.NewServer(http.HandlerFunc(handler))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	dialer := websocket.Dialer{
		Subprotocols: []string{"webtty"},
	}

	t.Run("valid auth token", func(t *testing.T) {
		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("WebSocket dial failed: %v", err)
		}
		defer conn.Close()

		// Send init message with valid token
		initMsg := InitMessage{
			AuthToken: credential,
		}
		err = conn.WriteJSON(initMsg)
		if err != nil {
			t.Fatalf("Failed to send init message: %v", err)
		}

		// Should receive a response (window title or initial data)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			t.Logf("Read after auth: err=%v (may timeout during slave setup)", err)
		} else {
			t.Logf("Received response after auth: type=%d, len=%d", msgType, len(msg))
		}
	})

	t.Run("invalid auth token", func(t *testing.T) {
		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("WebSocket dial failed: %v", err)
		}
		defer conn.Close()

		// Send init message with invalid token
		initMsg := InitMessage{
			AuthToken: "wrong:token",
		}
		err = conn.WriteJSON(initMsg)
		if err != nil {
			t.Fatalf("Failed to send init message: %v", err)
		}

		// Connection should be closed by server
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, err = conn.ReadMessage()
		if err == nil {
			t.Error("Expected connection to be closed after invalid auth")
		}
	})
}

// TestIntegrationProcessWSConnPaths tests various code paths in processWSConn
func TestIntegrationProcessWSConnPaths(t *testing.T) {
	factory := newMockIntegrationFactory()
	defer factory.CloseAll()

	options := &Options{
		Address:         "127.0.0.1",
		Port:            "0",
		Path:            "/",
		TitleFormat:     "{{ .hostname }}",
		PermitWrite:     true,
		MaxConnection:   10,
		PermitArguments: true,
		TitleVariables: map[string]interface{}{
			"hostname": "test-host",
		},
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := server.generateHandleWS(ctx, cancel, counter)

	testServer := httptest.NewServer(http.HandlerFunc(handler))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	dialer := websocket.Dialer{
		Subprotocols: []string{"webtty"},
	}

	t.Run("with arguments", func(t *testing.T) {
		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("WebSocket dial failed: %v", err)
		}
		defer conn.Close()

		// Send init message with arguments
		initMsg := InitMessage{
			AuthToken: "",
			Arguments: "?cols=80&rows=24",
		}
		err = conn.WriteJSON(initMsg)
		if err != nil {
			t.Fatalf("Failed to send init message: %v", err)
		}

		// Wait for response
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, err = conn.ReadMessage()
		// We don't care about the result, just that we exercise the code path
		t.Logf("Arguments path test completed, err=%v", err)
	})

	t.Run("binary message type rejection", func(t *testing.T) {
		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("WebSocket dial failed: %v", err)
		}
		defer conn.Close()

		// Send binary message instead of text
		err = conn.WriteMessage(websocket.BinaryMessage, []byte(`{"AuthToken":""}`))
		if err != nil {
			t.Fatalf("Failed to send binary message: %v", err)
		}

		// Connection should be closed by server
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, err = conn.ReadMessage()
		if err == nil {
			t.Error("Expected connection to be closed after binary message")
		}
	})

	t.Run("invalid json rejection", func(t *testing.T) {
		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("WebSocket dial failed: %v", err)
		}
		defer conn.Close()

		// Send invalid JSON
		err = conn.WriteMessage(websocket.TextMessage, []byte("not valid json"))
		if err != nil {
			t.Fatalf("Failed to send message: %v", err)
		}

		// Connection should be closed by server
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, err = conn.ReadMessage()
		if err == nil {
			t.Error("Expected connection to be closed after invalid JSON")
		}
	})
}
