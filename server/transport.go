package server

import (
	"io"
)

// Transport represents a bidirectional connection for terminal I/O.
// Both WebSocket and WebTransport implement this interface.
type Transport interface {
	io.ReadWriter
	Close() error
	RemoteAddr() string
}
