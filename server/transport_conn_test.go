package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

// connTestTransport implements the Transport interface for testing processTransportConn
type connTestTransport struct {
	readBuf    *bytes.Buffer
	writeBuf   *bytes.Buffer
	remoteAddr string
	closed     bool
	readErr    error
	writeErr   error
	mu         sync.Mutex
}

func newConnTestTransport() *connTestTransport {
	return &connTestTransport{
		readBuf:    new(bytes.Buffer),
		writeBuf:   new(bytes.Buffer),
		remoteAddr: "127.0.0.1:12345",
	}
}

func (m *connTestTransport) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.readErr != nil {
		return 0, m.readErr
	}
	return m.readBuf.Read(p)
}

func (m *connTestTransport) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.writeBuf.Write(p)
}

func (m *connTestTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *connTestTransport) RemoteAddr() string {
	return m.remoteAddr
}

func (m *connTestTransport) SetReadData(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readBuf = bytes.NewBuffer(data)
}

func (m *connTestTransport) SetReadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readErr = err
}

func (m *connTestTransport) GetWrittenData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeBuf.Bytes()
}

// mockSlaveForTransport implements the Slave interface for transport tests
type mockSlaveForTransport struct {
	reader     io.Reader
	writer     io.Writer
	closed     bool
	resizeFunc func(columns int, rows int) error
}

func newMockSlaveForTransport() *mockSlaveForTransport {
	r, w := io.Pipe()
	return &mockSlaveForTransport{
		reader: r,
		writer: w,
	}
}

func (m *mockSlaveForTransport) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

func (m *mockSlaveForTransport) Write(p []byte) (n int, err error) {
	return m.writer.Write(p)
}

func (m *mockSlaveForTransport) Close() error {
	m.closed = true
	return nil
}

func (m *mockSlaveForTransport) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{
		"command": "test",
	}
}

func (m *mockSlaveForTransport) ResizeTerminal(columns int, rows int) error {
	if m.resizeFunc != nil {
		return m.resizeFunc(columns, rows)
	}
	return nil
}

// connTestFactory implements Factory for transport tests
type connTestFactory struct {
	slave    *mockSlaveForTransport
	newError error
}

func newConnTestFactory() *connTestFactory {
	return &connTestFactory{
		slave: newMockSlaveForTransport(),
	}
}

func (m *connTestFactory) Name() string {
	return "mock-transport"
}

func (m *connTestFactory) New(params map[string][]string, headers map[string][]string) (Slave, error) {
	if m.newError != nil {
		return nil, m.newError
	}
	return m.slave, nil
}

func (m *connTestFactory) Command() (string, []string) {
	return "test", []string{}
}

func TestProcessTransportConnReadError(t *testing.T) {
	factory := newConnTestFactory()
	options := &Options{
		TitleFormat: "Test",
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	transport := newConnTestTransport()
	transport.SetReadError(errors.New("read error"))

	ctx := context.Background()
	err = server.processTransportConn(ctx, transport, nil)

	if err == nil {
		t.Error("processTransportConn should fail on read error")
	}
	if !containsString(err.Error(), "failed to read init message") {
		t.Errorf("Error should contain 'failed to read init message', got: %v", err)
	}
}

func TestProcessTransportConnInvalidJSON(t *testing.T) {
	factory := newConnTestFactory()
	options := &Options{
		TitleFormat: "Test",
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	transport := newConnTestTransport()
	transport.SetReadData([]byte("not valid json"))

	ctx := context.Background()
	err = server.processTransportConn(ctx, transport, nil)

	if err == nil {
		t.Error("processTransportConn should fail on invalid JSON")
	}
	if !containsString(err.Error(), "failed to parse init message") {
		t.Errorf("Error should contain 'failed to parse init message', got: %v", err)
	}
}

func TestProcessTransportConnAuthFailure(t *testing.T) {
	factory := newConnTestFactory()
	options := &Options{
		TitleFormat: "Test",
		Credential:  "correct:password",
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	transport := newConnTestTransport()
	initMsg := InitMessage{AuthToken: "wrong:password"}
	data, _ := json.Marshal(initMsg)
	transport.SetReadData(data)

	ctx := context.Background()
	err = server.processTransportConn(ctx, transport, nil)

	if err == nil {
		t.Error("processTransportConn should fail on auth failure")
	}
	if !containsString(err.Error(), "authentication failed") {
		t.Errorf("Error should contain 'authentication failed', got: %v", err)
	}
}

func TestProcessTransportConnFactoryError(t *testing.T) {
	factory := newConnTestFactory()
	factory.newError = errors.New("factory error")

	options := &Options{
		TitleFormat: "Test",
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	transport := newConnTestTransport()
	initMsg := InitMessage{AuthToken: ""}
	data, _ := json.Marshal(initMsg)
	transport.SetReadData(data)

	ctx := context.Background()
	err = server.processTransportConn(ctx, transport, nil)

	if err == nil {
		t.Error("processTransportConn should fail on factory error")
	}
	if !containsString(err.Error(), "failed to create backend") {
		t.Errorf("Error should contain 'failed to create backend', got: %v", err)
	}
}

func TestProcessTransportConnWithArguments(t *testing.T) {
	factory := newConnTestFactory()
	options := &Options{
		TitleFormat:     "Test",
		PermitArguments: true,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	transport := newConnTestTransport()
	initMsg := InitMessage{
		AuthToken: "",
		Arguments: "?cols=80&rows=24",
	}
	data, _ := json.Marshal(initMsg)
	transport.SetReadData(data)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This will timeout because webtty.Run will wait for more data
	err = server.processTransportConn(ctx, transport, nil)

	// Error is expected due to context cancellation or EOF
	t.Logf("processTransportConn result: %v (expected due to timeout)", err)
}

func TestProcessTransportConnInvalidArguments(t *testing.T) {
	factory := newConnTestFactory()
	options := &Options{
		TitleFormat:     "Test",
		PermitArguments: true,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	transport := newConnTestTransport()
	initMsg := InitMessage{
		AuthToken: "",
		Arguments: "://invalid-url", // Invalid URL
	}
	data, _ := json.Marshal(initMsg)
	transport.SetReadData(data)

	ctx := context.Background()
	err = server.processTransportConn(ctx, transport, nil)

	if err == nil {
		t.Error("processTransportConn should fail on invalid arguments URL")
	}
	if !containsString(err.Error(), "failed to parse arguments") {
		t.Errorf("Error should contain 'failed to parse arguments', got: %v", err)
	}
}

func TestProcessTransportConnWithOptions(t *testing.T) {
	factory := newConnTestFactory()
	options := &Options{
		TitleFormat:     "{{ .hostname }}",
		PermitWrite:     true,
		EnableReconnect: true,
		ReconnectTime:   10,
		Width:           80,
		Height:          24,
		TitleVariables: map[string]interface{}{
			"hostname": "test-host",
		},
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	transport := newConnTestTransport()
	initMsg := InitMessage{AuthToken: ""}
	data, _ := json.Marshal(initMsg)
	transport.SetReadData(data)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This will timeout because webtty.Run will wait for more data
	err = server.processTransportConn(ctx, transport, nil)

	// Error is expected due to context cancellation or EOF
	t.Logf("processTransportConn with options result: %v (expected)", err)
}

// containsString checks if the string s contains the substring substr
func containsString(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
