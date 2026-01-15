package server

import (
	"testing"
)

// mockFactory is a mock implementation of Factory for testing
type mockFactory struct {
	name    string
	command string
	argv    []string
}

func newMockFactory() *mockFactory {
	return &mockFactory{
		name:    "mock",
		command: "/bin/bash",
		argv:    []string{"-c", "echo test"},
	}
}

func (m *mockFactory) Name() string {
	return m.name
}

func (m *mockFactory) New(params map[string][]string, headers map[string][]string) (Slave, error) {
	return nil, nil
}

func (m *mockFactory) Command() (string, []string) {
	return m.command, m.argv
}

func TestNewServer(t *testing.T) {
	factory := newMockFactory()
	options := &Options{
		TitleFormat: "WebTmux",
	}

	server, err := New(factory, options)

	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if server == nil {
		t.Fatal("New() returned nil server")
	}
	if server.factory != factory {
		t.Error("server.factory mismatch")
	}
	if server.options != options {
		t.Error("server.options mismatch")
	}
	if server.upgrader == nil {
		t.Error("server.upgrader is nil")
	}
	if server.indexTemplate == nil {
		t.Error("server.indexTemplate is nil")
	}
	if server.titleTemplate == nil {
		t.Error("server.titleTemplate is nil")
	}
	if server.manifestTemplate == nil {
		t.Error("server.manifestTemplate is nil")
	}
}

func TestNewServerWithWSOrigin(t *testing.T) {
	factory := newMockFactory()
	options := &Options{
		TitleFormat: "WebTmux",
		WSOrigin:    `https://example\.com`,
	}

	server, err := New(factory, options)

	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if server == nil {
		t.Fatal("New() returned nil server")
	}
	if server.upgrader.CheckOrigin == nil {
		t.Error("CheckOrigin should be set for WSOrigin")
	}
}

func TestNewServerInvalidWSOrigin(t *testing.T) {
	factory := newMockFactory()
	options := &Options{
		TitleFormat: "WebTmux",
		WSOrigin:    "[invalid regex",
	}

	_, err := New(factory, options)

	if err == nil {
		t.Error("New() should fail with invalid WSOrigin regex")
	}
}

func TestNewServerInvalidTitleFormat(t *testing.T) {
	factory := newMockFactory()
	options := &Options{
		TitleFormat: "{{ .invalid",
	}

	_, err := New(factory, options)

	if err == nil {
		t.Error("New() should fail with invalid TitleFormat template")
	}
}

func TestDetectTmuxSession(t *testing.T) {
	tests := []struct {
		name    string
		command string
		argv    []string
		want    string
	}{
		{
			name:    "not tmux command",
			command: "/bin/bash",
			argv:    []string{"-c", "echo test"},
			want:    "",
		},
		{
			name:    "tmux attach with -t flag",
			command: "tmux",
			argv:    []string{"attach", "-t", "mysession"},
			want:    "mysession",
		},
		{
			name:    "tmux attach-session with -t flag",
			command: "tmux",
			argv:    []string{"attach-session", "-t", "dev"},
			want:    "dev",
		},
		{
			name:    "tmux a with -t flag",
			command: "tmux",
			argv:    []string{"a", "-t", "test"},
			want:    "test",
		},
		{
			name:    "tmux attach without session",
			command: "tmux",
			argv:    []string{"attach"},
			want:    "0", // defaults to "0" when no session specified
		},
		{
			name:    "tmux with other command",
			command: "tmux",
			argv:    []string{"list-sessions"},
			want:    "0", // defaults to "0" when no session specified
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &mockFactory{
				name:    "mock",
				command: tt.command,
				argv:    tt.argv,
			}
			options := &Options{
				TitleFormat: "test",
			}

			server, err := New(factory, options)
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}

			if server.tmuxSession != tt.want {
				t.Errorf("tmuxSession = %q, want %q", server.tmuxSession, tt.want)
			}
		})
	}
}

func TestNewServerWithCustomIndexFile(t *testing.T) {
	factory := newMockFactory()
	options := &Options{
		TitleFormat: "WebTmux",
		IndexFile:   "/nonexistent/index.html",
	}

	_, err := New(factory, options)

	if err == nil {
		t.Error("New() should fail with nonexistent IndexFile")
	}
}

// Benchmark server creation
func BenchmarkNewServer(b *testing.B) {
	factory := newMockFactory()
	options := &Options{
		TitleFormat: "WebTmux",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New(factory, options)
	}
}
