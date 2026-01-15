package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSetupHTTPServerBasic(t *testing.T) {
	factory := newMockFactory()
	options := &Options{
		TitleFormat: "Test",
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv, err := server.setupHTTPServer(handler)
	if err != nil {
		t.Fatalf("setupHTTPServer() error: %v", err)
	}

	if srv == nil {
		t.Fatal("setupHTTPServer() returned nil")
	}

	if srv.Handler == nil {
		t.Error("srv.Handler is nil")
	}

	// TLS should not be configured when EnableTLSClientAuth is false
	if srv.TLSConfig != nil {
		t.Error("TLSConfig should be nil when EnableTLSClientAuth is false")
	}
}

func TestSetupHTTPServerWithTLSClientAuth(t *testing.T) {
	// Generate test CA
	caFile, cleanup := generateCAForHTTPTest(t)
	defer cleanup()

	factory := newMockFactory()
	options := &Options{
		TitleFormat:         "Test",
		EnableTLSClientAuth: true,
		TLSCACrtFile:        caFile,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv, err := server.setupHTTPServer(handler)
	if err != nil {
		t.Fatalf("setupHTTPServer() error: %v", err)
	}

	if srv == nil {
		t.Fatal("setupHTTPServer() returned nil")
	}

	// TLS should be configured
	if srv.TLSConfig == nil {
		t.Error("TLSConfig should be set when EnableTLSClientAuth is true")
	}
}

func TestSetupHTTPServerTLSClientAuthError(t *testing.T) {
	factory := newMockFactory()
	options := &Options{
		TitleFormat:         "Test",
		EnableTLSClientAuth: true,
		TLSCACrtFile:        "/nonexistent/ca.pem",
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_, err = server.setupHTTPServer(handler)
	if err == nil {
		t.Error("setupHTTPServer() should fail with nonexistent CA file")
	}
}

// generateCAForHTTPTest creates a self-signed CA certificate for testing
func generateCAForHTTPTest(t *testing.T) (certFile string, cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "webtmux-http-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Generate CA private key
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to generate CA key: %v", err)
	}

	// Create CA certificate template
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"WebTmux HTTP Test CA"},
			CommonName:   "HTTP Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create CA certificate: %v", err)
	}

	// Write CA certificate to file
	certFile = filepath.Join(tmpDir, "ca.pem")
	certOut, err := os.Create(certFile)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create cert file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER}); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write cert: %v", err)
	}

	cleanup = func() {
		os.RemoveAll(tmpDir)
	}

	return certFile, cleanup
}

func TestSetupHTTPServerWithInvalidTLSCert(t *testing.T) {
	// Create temp file with invalid certificate data
	tmpDir, err := os.MkdirTemp("", "webtmux-http-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	caFile := filepath.Join(tmpDir, "invalid-ca.pem")
	if err := os.WriteFile(caFile, []byte("not a valid certificate"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	factory := newMockFactory()
	options := &Options{
		TitleFormat:         "Test",
		EnableTLSClientAuth: true,
		TLSCACrtFile:        caFile,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_, err = server.setupHTTPServer(handler)
	if err == nil {
		t.Error("setupHTTPServer() should fail with invalid CA certificate")
	}
}

// Benchmark setupHTTPServer
func BenchmarkSetupHTTPServer(b *testing.B) {
	factory := newMockFactory()
	options := &Options{
		TitleFormat: "Test",
	}

	server, _ := New(factory, options)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.setupHTTPServer(handler)
	}
}
