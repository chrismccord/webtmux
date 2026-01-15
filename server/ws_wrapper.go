package server

import (
	"io"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// wsTransport wraps a WebSocket connection to implement the Transport interface.
type wsTransport struct {
	*websocket.Conn
}

// Write sends data over the WebSocket connection as a TextMessage.
func (wst *wsTransport) Write(p []byte) (n int, err error) {
	writer, err := wst.Conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}
	defer writer.Close()
	return writer.Write(p)
}

// Read reads data from the WebSocket connection, only accepting TextMessages.
func (wst *wsTransport) Read(p []byte) (n int, err error) {
	for {
		msgType, reader, err := wst.Conn.NextReader()
		if err != nil {
			return 0, err
		}

		if msgType != websocket.TextMessage {
			continue
		}

		b, err := io.ReadAll(reader)
		if err != nil {
			return 0, err
		}
		if len(b) > len(p) {
			return 0, errors.New("client message exceeded buffer size")
		}
		n = copy(p, b)
		return n, nil
	}
}

// Close closes the WebSocket connection.
func (wst *wsTransport) Close() error {
	return wst.Conn.Close()
}

// RemoteAddr returns the remote address of the WebSocket connection.
func (wst *wsTransport) RemoteAddr() string {
	return wst.Conn.RemoteAddr().String()
}

// Ensure wsTransport implements Transport interface
var _ Transport = (*wsTransport)(nil)
