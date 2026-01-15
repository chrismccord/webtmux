package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// generateTestCA creates a self-signed CA certificate for testing
func generateTestCA(t *testing.T) (certFile string, cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "webtmux-tls-test-*")
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
			Organization: []string{"WebTmux Test CA"},
			CommonName:   "Test CA",
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

func TestTLSConfig(t *testing.T) {
	caFile, cleanup := generateTestCA(t)
	defer cleanup()

	factory := newMockFactory()
	options := &Options{
		TitleFormat:  "test",
		TLSCACrtFile: caFile,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	tlsConfig, err := server.tlsConfig()
	if err != nil {
		t.Fatalf("tlsConfig() error: %v", err)
	}

	if tlsConfig == nil {
		t.Fatal("tlsConfig() returned nil")
	}

	if tlsConfig.ClientAuth != 4 { // tls.RequireAndVerifyClientCert
		t.Errorf("ClientAuth = %v, want RequireAndVerifyClientCert", tlsConfig.ClientAuth)
	}

	if tlsConfig.ClientCAs == nil {
		t.Error("ClientCAs is nil")
	}
}

func TestTLSConfigFileNotFound(t *testing.T) {
	factory := newMockFactory()
	options := &Options{
		TitleFormat:  "test",
		TLSCACrtFile: "/nonexistent/ca.pem",
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	_, err = server.tlsConfig()
	if err == nil {
		t.Error("tlsConfig() should fail with nonexistent file")
	}
}

func TestTLSConfigInvalidCert(t *testing.T) {
	// Create temp file with invalid certificate data
	tmpDir, err := os.MkdirTemp("", "webtmux-tls-test-*")
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
		TitleFormat:  "test",
		TLSCACrtFile: caFile,
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	_, err = server.tlsConfig()
	if err == nil {
		t.Error("tlsConfig() should fail with invalid certificate")
	}
}

func TestTLSConfigWithHomeDir(t *testing.T) {
	caFile, cleanup := generateTestCA(t)
	defer cleanup()

	factory := newMockFactory()
	options := &Options{
		TitleFormat:  "test",
		TLSCACrtFile: caFile, // Use absolute path
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	tlsConfig, err := server.tlsConfig()
	if err != nil {
		t.Fatalf("tlsConfig() error: %v", err)
	}

	if tlsConfig == nil {
		t.Fatal("tlsConfig() returned nil")
	}
}
