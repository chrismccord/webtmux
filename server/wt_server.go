package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// WebTransportServer handles WebTransport connections over HTTP/3.
type WebTransportServer struct {
	server       *webtransport.Server
	options      *Options
	pathPrefix   string
	originRegexp *regexp.Regexp
}

// NewWebTransportServer creates a new WebTransport server.
func NewWebTransportServer(options *Options, pathPrefix string) (*WebTransportServer, error) {
	var originRegexp *regexp.Regexp
	var err error

	if options.WSOrigin != "" {
		originRegexp, err = regexp.Compile(options.WSOrigin)
		if err != nil {
			return nil, fmt.Errorf("failed to compile origin regex: %w", err)
		}
	}

	addr := fmt.Sprintf("%s:%s", options.Address, options.Port)

	wtServer := &webtransport.Server{
		H3: &http3.Server{
			Addr: addr,
		},
		CheckOrigin: func(r *http.Request) bool {
			if originRegexp != nil {
				return originRegexp.MatchString(r.Header.Get("Origin"))
			}
			// Default: allow all origins (auth provides protection)
			return true
		},
	}

	return &WebTransportServer{
		server:       wtServer,
		options:      options,
		pathPrefix:   pathPrefix,
		originRegexp: originRegexp,
	}, nil
}

// Upgrade upgrades an HTTP request to a WebTransport session.
func (wts *WebTransportServer) Upgrade(w http.ResponseWriter, r *http.Request) (*webtransport.Session, error) {
	return wts.server.Upgrade(w, r)
}

// ListenAndServeTLS starts the WebTransport server with TLS.
func (wts *WebTransportServer) ListenAndServeTLS(ctx context.Context, certFile, keyFile string, handler http.Handler) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	wts.server.H3.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
	}

	wts.server.H3.Handler = handler

	log.Printf("WebTransport server listening on %s:%s (UDP)", wts.options.Address, wts.options.Port)

	// Run in a goroutine and handle context cancellation
	errChan := make(chan error, 1)
	go func() {
		errChan <- wts.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		wts.Close()
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// Close shuts down the WebTransport server.
func (wts *WebTransportServer) Close() error {
	return wts.server.Close()
}

// Server returns the underlying webtransport.Server for direct access.
func (wts *WebTransportServer) Server() *webtransport.Server {
	return wts.server
}
