package server

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLogResponseWriterDefaultStatus(t *testing.T) {
	rr := httptest.NewRecorder()
	lrw := &logResponseWriter{ResponseWriter: rr, status: 200}

	if lrw.status != 200 {
		t.Errorf("status = %d, want 200", lrw.status)
	}
}

func TestLogResponseWriterWriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	lrw := &logResponseWriter{ResponseWriter: rr, status: 200}

	lrw.WriteHeader(http.StatusNotFound)

	if lrw.status != http.StatusNotFound {
		t.Errorf("status = %d, want %d", lrw.status, http.StatusNotFound)
	}
	if rr.Code != http.StatusNotFound {
		t.Errorf("underlying recorder Code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestLogResponseWriterWrite(t *testing.T) {
	rr := httptest.NewRecorder()
	lrw := &logResponseWriter{ResponseWriter: rr, status: 200}

	data := []byte("test response")
	n, err := lrw.Write(data)

	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() returned %d, want %d", n, len(data))
	}
	if rr.Body.String() != "test response" {
		t.Errorf("Body = %s, want 'test response'", rr.Body.String())
	}
}

// mockHijackResponseWriter implements http.Hijacker for testing
type mockHijackResponseWriter struct {
	http.ResponseWriter
	hijacked bool
}

func (m *mockHijackResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	// Return mock connection - using nil for simplicity in test
	return &mockConn{}, bufio.NewReadWriter(bufio.NewReader(nil), bufio.NewWriter(nil)), nil
}

type mockConn struct{}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestLogResponseWriterHijack(t *testing.T) {
	mrw := &mockHijackResponseWriter{ResponseWriter: httptest.NewRecorder()}
	lrw := &logResponseWriter{ResponseWriter: mrw, status: 200}

	conn, _, err := lrw.Hijack()

	if err != nil {
		t.Fatalf("Hijack() error: %v", err)
	}
	if conn == nil {
		t.Error("Hijack() returned nil connection")
	}
	if lrw.status != http.StatusSwitchingProtocols {
		t.Errorf("status after Hijack = %d, want %d", lrw.status, http.StatusSwitchingProtocols)
	}
	if !mrw.hijacked {
		t.Error("Underlying ResponseWriter.Hijack() was not called")
	}
}

func TestLogResponseWriterMultipleHeaders(t *testing.T) {
	rr := httptest.NewRecorder()
	lrw := &logResponseWriter{ResponseWriter: rr, status: 200}

	// Set various headers
	lrw.Header().Set("Content-Type", "text/plain")
	lrw.Header().Set("X-Custom-Header", "value")

	if lrw.Header().Get("Content-Type") != "text/plain" {
		t.Error("Content-Type header not set correctly")
	}
	if lrw.Header().Get("X-Custom-Header") != "value" {
		t.Error("X-Custom-Header not set correctly")
	}
}
