package server

import (
	"bytes"
	"io"
	"testing"
)

// mockTransport is a simple mock implementation of Transport for testing
type mockTransport struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
	addr     string
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		readBuf:  bytes.NewBuffer(nil),
		writeBuf: bytes.NewBuffer(nil),
		addr:     "127.0.0.1:12345",
	}
}

func (m *mockTransport) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.EOF
	}
	return m.readBuf.Read(p)
}

func (m *mockTransport) Write(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.writeBuf.Write(p)
}

func (m *mockTransport) Close() error {
	m.closed = true
	return nil
}

func (m *mockTransport) RemoteAddr() string {
	return m.addr
}

// Verify mockTransport implements Transport interface
var _ Transport = (*mockTransport)(nil)

func TestMockTransportImplementsInterface(t *testing.T) {
	mt := newMockTransport()

	// Test Write
	data := []byte("hello world")
	n, err := mt.Write(data)
	if err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() returned %d, expected %d", n, len(data))
	}

	// Verify written data
	written := mt.writeBuf.Bytes()
	if !bytes.Equal(written, data) {
		t.Errorf("Written data = %v, expected %v", written, data)
	}

	// Test RemoteAddr
	addr := mt.RemoteAddr()
	if addr != "127.0.0.1:12345" {
		t.Errorf("RemoteAddr() = %s, expected 127.0.0.1:12345", addr)
	}

	// Test Read
	mt.readBuf.Write([]byte("response"))
	buf := make([]byte, 100)
	n, err = mt.Read(buf)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}
	if n != 8 {
		t.Errorf("Read() returned %d bytes, expected 8", n)
	}
	if !bytes.Equal(buf[:n], []byte("response")) {
		t.Errorf("Read data = %v, expected 'response'", buf[:n])
	}

	// Test Close
	err = mt.Close()
	if err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}
	if !mt.closed {
		t.Error("Transport not marked as closed")
	}

	// Test operations after close
	_, err = mt.Read(buf)
	if err != io.EOF {
		t.Errorf("Read after close should return EOF, got: %v", err)
	}

	_, err = mt.Write(data)
	if err != io.ErrClosedPipe {
		t.Errorf("Write after close should return ErrClosedPipe, got: %v", err)
	}
}

func TestTransportReadWriter(t *testing.T) {
	mt := newMockTransport()

	// Verify Transport satisfies io.ReadWriter
	var rw io.ReadWriter = mt
	_ = rw

	// Test io.ReadWriter usage
	testData := []byte("test message")
	_, err := io.WriteString(mt, string(testData))
	if err != nil {
		t.Fatalf("io.WriteString() failed: %v", err)
	}

	if !bytes.Equal(mt.writeBuf.Bytes(), testData) {
		t.Errorf("io.WriteString wrote %v, expected %v", mt.writeBuf.Bytes(), testData)
	}
}
