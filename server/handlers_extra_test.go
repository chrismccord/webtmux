package server

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	texttemplate "text/template"
	"time"
)

func TestHandleTmuxEventsNilController(t *testing.T) {
	factory := newMockFactory()
	options := &Options{
		TitleFormat: "Test",
	}

	server, err := New(factory, options)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Ensure tmuxCtrl is nil
	server.tmuxCtrl = nil

	ctx := context.Background()

	// Should return immediately without blocking
	done := make(chan struct{})
	go func() {
		server.handleTmuxEvents(ctx, nil)
		close(done)
	}()

	// Give the goroutine time to complete
	select {
	case <-done:
		// Function returned immediately as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("handleTmuxEvents should return immediately when tmuxCtrl is nil")
	}
}

func TestHandleIndexError(t *testing.T) {
	// Test with a title template that will fail (using missingkey=error)
	badTitleTmpl, _ := texttemplate.New("title").Option("missingkey=error").Parse("{{ .nonexistent }}")
	indexTmpl, _ := template.New("index").Parse("<html>{{ .title }}</html>")

	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{},
		},
		titleTemplate: badTitleTmpl,
		indexTemplate: indexTmpl,
	}

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	server.handleIndex(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("handleIndex() status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleManifestError(t *testing.T) {
	// Test with a title template that will fail (using missingkey=error)
	badTitleTmpl, _ := texttemplate.New("title").Option("missingkey=error").Parse("{{ .nonexistent }}")
	manifestTmpl, _ := template.New("manifest").Parse(`{"name": "{{ .title }}"}`)

	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{},
		},
		titleTemplate:    badTitleTmpl,
		manifestTemplate: manifestTmpl,
	}

	req := httptest.NewRequest("GET", "/manifest.webmanifest", nil)
	rr := httptest.NewRecorder()

	server.handleManifest(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("handleManifest() status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleIndexTemplateError(t *testing.T) {
	// Test with an index template that will fail (using missingkey=error)
	titleTmpl, _ := texttemplate.New("title").Parse("Test")
	badIndexTmpl, _ := template.New("index").Option("missingkey=error").Parse("{{ .missing }}")

	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{},
		},
		titleTemplate: titleTmpl,
		indexTemplate: badIndexTmpl,
	}

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	server.handleIndex(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("handleIndex() status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleManifestTemplateError(t *testing.T) {
	// Test with a manifest template that will fail (using missingkey=error)
	titleTmpl, _ := texttemplate.New("title").Parse("Test")
	badManifestTmpl, _ := template.New("manifest").Option("missingkey=error").Parse("{{ .missing }}")

	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{},
		},
		titleTemplate:    titleTmpl,
		manifestTemplate: badManifestTmpl,
	}

	req := httptest.NewRequest("GET", "/manifest.webmanifest", nil)
	rr := httptest.NewRecorder()

	server.handleManifest(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("handleManifest() status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestIndexVariablesWithRemoteAddr(t *testing.T) {
	titleTmpl, _ := texttemplate.New("title").Parse("{{ .remote_addr }}")

	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{},
		},
		titleTemplate: titleTmpl,
	}

	// Test with X-Forwarded-For header
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.RemoteAddr = "10.0.0.1:12345"

	vars, err := server.indexVariables(req)
	if err != nil {
		t.Fatalf("indexVariables() error: %v", err)
	}

	// The remote_addr should use X-Forwarded-For when present
	title, ok := vars["title"].(string)
	if !ok {
		t.Fatal("title should be a string")
	}

	// Title should contain some address
	if title == "" {
		t.Error("title should not be empty")
	}
}

func TestIndexVariablesRemoteAddrParsing(t *testing.T) {
	titleTmpl, _ := texttemplate.New("title").Parse("{{ .remote_addr }}")

	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{},
		},
		titleTemplate: titleTmpl,
	}

	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		expectedResult bool
	}{
		{
			name:           "standard IP:port",
			remoteAddr:     "192.168.1.1:12345",
			xForwardedFor:  "",
			expectedResult: true,
		},
		{
			name:           "with X-Forwarded-For",
			remoteAddr:     "10.0.0.1:8080",
			xForwardedFor:  "203.0.113.50",
			expectedResult: true,
		},
		{
			name:           "IPv6 address",
			remoteAddr:     "[::1]:12345",
			xForwardedFor:  "",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			vars, err := server.indexVariables(req)
			if tt.expectedResult && err != nil {
				t.Fatalf("indexVariables() error: %v", err)
			}
			if !tt.expectedResult && err == nil {
				t.Fatal("indexVariables() should have failed")
			}
			if tt.expectedResult {
				if vars["title"] == nil {
					t.Error("title should be set")
				}
			}
		})
	}
}
