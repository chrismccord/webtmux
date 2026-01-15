package server

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	texttemplate "text/template"
)

func TestHandleConfig(t *testing.T) {
	server := &Server{
		options: &Options{
			EnableWebTransport: true,
			WebTransportPort:   "8443",
			WSQueryArgs:        "test=1",
		},
	}

	req := httptest.NewRequest("GET", "/config.js", nil)
	rr := httptest.NewRecorder()

	server.handleConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleConfig() status = %d, want %d", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/javascript" {
		t.Errorf("Content-Type = %s, want 'application/javascript'", contentType)
	}

	body := rr.Body.String()

	// Check for expected config values
	if !strings.Contains(body, "gotty_term = 'xterm'") {
		t.Error("Config should contain gotty_term = 'xterm'")
	}
	if !strings.Contains(body, "gotty_ws_query_args = 'test=1'") {
		t.Error("Config should contain ws_query_args")
	}
	if !strings.Contains(body, "gotty_webtransport_enabled = true") {
		t.Error("Config should contain webtransport_enabled = true")
	}
	if !strings.Contains(body, "gotty_webtransport_port = '8443'") {
		t.Error("Config should contain webtransport_port = '8443'")
	}
}

func TestHandleConfigWebTransportDisabled(t *testing.T) {
	server := &Server{
		options: &Options{
			EnableWebTransport: false,
			WebTransportPort:   "",
			WSQueryArgs:        "",
		},
	}

	req := httptest.NewRequest("GET", "/config.js", nil)
	rr := httptest.NewRecorder()

	server.handleConfig(rr, req)

	body := rr.Body.String()

	if !strings.Contains(body, "gotty_webtransport_enabled = false") {
		t.Error("Config should contain webtransport_enabled = false")
	}
}

func TestHandleAuthToken(t *testing.T) {
	server := &Server{
		options: &Options{
			Credential: "admin:secret",
		},
	}

	req := httptest.NewRequest("GET", "/auth_token.js", nil)
	rr := httptest.NewRecorder()

	server.handleAuthToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleAuthToken() status = %d, want %d", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/javascript" {
		t.Errorf("Content-Type = %s, want 'application/javascript'", contentType)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "gotty_auth_token") {
		t.Error("Response should contain gotty_auth_token")
	}
	if !strings.Contains(body, "admin:secret") {
		t.Error("Response should contain the credential")
	}
}

func TestTitleVariables(t *testing.T) {
	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{
				"hostname": "test-server",
			},
		},
	}

	order := []string{"server", "master"}
	varUnits := map[string]map[string]interface{}{
		"server": {"hostname": "test-server"},
		"master": {"remote_addr": "192.168.1.1"},
	}

	result := server.titleVariables(order, varUnits)

	if result["hostname"] != "test-server" {
		t.Errorf("hostname = %v, want 'test-server'", result["hostname"])
	}
	if result["remote_addr"] != "192.168.1.1" {
		t.Errorf("remote_addr = %v, want '192.168.1.1'", result["remote_addr"])
	}
}

func TestTitleVariablesOverride(t *testing.T) {
	server := &Server{
		options: &Options{},
	}

	// Test that later entries in order override earlier ones
	order := []string{"first", "second"}
	varUnits := map[string]map[string]interface{}{
		"first":  {"key": "value1"},
		"second": {"key": "value2"},
	}

	result := server.titleVariables(order, varUnits)

	// Second should override first
	if result["key"] != "value2" {
		t.Errorf("key = %v, want 'value2' (should be overridden)", result["key"])
	}
}

func TestTitleVariablesPanic(t *testing.T) {
	server := &Server{
		options: &Options{},
	}

	order := []string{"nonexistent"}
	varUnits := map[string]map[string]interface{}{
		"existing": {"key": "value"},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("titleVariables should panic with nonexistent key in order")
		}
	}()

	server.titleVariables(order, varUnits)
}

func TestIndexVariables(t *testing.T) {
	titleTmpl, _ := texttemplate.New("title").Parse("WebTmux - {{ .hostname }}")
	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{
				"hostname": "test-host",
			},
		},
		titleTemplate: titleTmpl,
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	vars, err := server.indexVariables(req)

	if err != nil {
		t.Fatalf("indexVariables() error: %v", err)
	}

	if vars["title"] == nil {
		t.Error("title should be set in indexVariables result")
	}

	title, ok := vars["title"].(string)
	if !ok {
		t.Error("title should be a string")
	}
	if !strings.Contains(title, "test-host") {
		t.Errorf("title = %s, should contain 'test-host'", title)
	}
}

func TestHandleIndex(t *testing.T) {
	titleTmpl, _ := texttemplate.New("title").Parse("Test Title")
	indexTmpl, _ := template.New("index").Parse("<html><title>{{ .title }}</title></html>")

	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{},
		},
		titleTemplate: titleTmpl,
		indexTemplate: indexTmpl,
	}

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	server.handleIndex(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleIndex() status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "<html>") {
		t.Error("Response should contain HTML")
	}
	if !strings.Contains(body, "Test Title") {
		t.Error("Response should contain the title")
	}
}

func TestHandleManifest(t *testing.T) {
	titleTmpl, _ := texttemplate.New("title").Parse("WebTmux")
	manifestTmpl, _ := template.New("manifest").Parse(`{"name": "{{ .title }}"}`)

	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{},
		},
		titleTemplate:    titleTmpl,
		manifestTemplate: manifestTmpl,
	}

	req := httptest.NewRequest("GET", "/manifest.webmanifest", nil)
	rr := httptest.NewRecorder()

	server.handleManifest(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleManifest() status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "WebTmux") {
		t.Error("Response should contain manifest data")
	}
}

func TestHandleIndexWithComplexTitle(t *testing.T) {
	// Test with complex title variables
	titleTmpl, _ := texttemplate.New("title").Parse("{{ .hostname }} - {{ .remote_addr }}")
	indexTmpl, _ := template.New("index").Parse("<html><title>{{ .title }}</title></html>")

	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{
				"hostname": "webserver",
			},
		},
		titleTemplate: titleTmpl,
		indexTemplate: indexTmpl,
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:8080"
	rr := httptest.NewRecorder()

	server.handleIndex(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleIndex() status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "webserver") {
		t.Error("Response should contain hostname")
	}
}

func TestHandleManifestWithComplexTitle(t *testing.T) {
	// Test with complex title variables
	titleTmpl, _ := texttemplate.New("title").Parse("{{ .hostname }}")
	manifestTmpl, _ := template.New("manifest").Parse(`{"name": "{{ .title }}", "short_name": "WTM"}`)

	server := &Server{
		options: &Options{
			TitleVariables: map[string]interface{}{
				"hostname": "my-terminal",
			},
		},
		titleTemplate:    titleTmpl,
		manifestTemplate: manifestTmpl,
	}

	req := httptest.NewRequest("GET", "/manifest.webmanifest", nil)
	rr := httptest.NewRecorder()

	server.handleManifest(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleManifest() status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "my-terminal") {
		t.Error("Response should contain hostname in manifest")
	}
}

// Benchmark handler functions
func BenchmarkHandleConfig(b *testing.B) {
	server := &Server{
		options: &Options{
			EnableWebTransport: true,
			WebTransportPort:   "8443",
			WSQueryArgs:        "test=1",
		},
	}

	req := httptest.NewRequest("GET", "/config.js", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		server.handleConfig(rr, req)
	}
}

func BenchmarkTitleVariables(b *testing.B) {
	server := &Server{
		options: &Options{},
	}

	order := []string{"server", "master"}
	varUnits := map[string]map[string]interface{}{
		"server": {"hostname": "test"},
		"master": {"addr": "127.0.0.1"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.titleVariables(order, varUnits)
	}
}
