package server

import (
	"encoding/binary"
	"io"
	"sync"

	"github.com/pkg/errors"
	"github.com/quic-go/webtransport-go"
)

// wtTransport wraps a WebTransport bidirectional stream to implement the Transport interface.
// It uses length-prefixed framing to match WebSocket's message semantics.
type wtTransport struct {
	session *webtransport.Session
	stream  *webtransport.Stream
	mu      sync.Mutex
}

// newWTTransport creates a new WebTransport transport wrapper.
func newWTTransport(session *webtransport.Session, stream *webtransport.Stream) *wtTransport {
	return &wtTransport{
		session: session,
		stream:  stream,
	}
}

// Write sends data over the WebTransport stream with length-prefixed framing.
// Format: [2-byte big-endian length][payload]
func (wtt *wtTransport) Write(p []byte) (n int, err error) {
	wtt.mu.Lock()
	defer wtt.mu.Unlock()

	if len(p) > 65535 {
		return 0, errors.New("message too large for WebTransport frame (max 65535 bytes)")
	}

	// Write length prefix (2 bytes, big-endian)
	header := make([]byte, 2)
	binary.BigEndian.PutUint16(header, uint16(len(p)))

	if _, err := wtt.stream.Write(header); err != nil {
		return 0, errors.Wrap(err, "failed to write frame header")
	}

	// Write payload
	written, err := wtt.stream.Write(p)
	if err != nil {
		return written, errors.Wrap(err, "failed to write frame payload")
	}

	return written, nil
}

// Read reads a length-prefixed frame from the WebTransport stream.
func (wtt *wtTransport) Read(p []byte) (n int, err error) {
	// Read length prefix (2 bytes)
	header := make([]byte, 2)
	if _, err := io.ReadFull(wtt.stream, header); err != nil {
		return 0, err
	}

	length := int(binary.BigEndian.Uint16(header))
	if length > len(p) {
		return 0, errors.Errorf("message size %d exceeds buffer size %d", length, len(p))
	}

	// Read payload
	return io.ReadFull(wtt.stream, p[:length])
}

// Close closes the WebTransport stream and session.
func (wtt *wtTransport) Close() error {
	var err error
	if wtt.stream != nil {
		err = wtt.stream.Close()
	}
	if wtt.session != nil {
		wtt.session.CloseWithError(0, "connection closed")
	}
	return err
}

// RemoteAddr returns the remote address of the WebTransport session.
func (wtt *wtTransport) RemoteAddr() string {
	if wtt.session != nil {
		return wtt.session.RemoteAddr().String()
	}
	return "unknown"
}

// Ensure wtTransport implements Transport interface
var _ Transport = (*wtTransport)(nil)
