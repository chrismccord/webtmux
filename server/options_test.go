package server

import (
	"testing"
)

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		options *Options
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid options - defaults",
			options: &Options{
				EnableTLS:            false,
				EnableTLSClientAuth:  false,
				EnableWebTransport:   false,
			},
			wantErr: false,
		},
		{
			name: "valid options - TLS enabled",
			options: &Options{
				EnableTLS:            true,
				EnableTLSClientAuth:  false,
				EnableWebTransport:   false,
			},
			wantErr: false,
		},
		{
			name: "valid options - TLS with client auth",
			options: &Options{
				EnableTLS:           true,
				EnableTLSClientAuth: true,
				EnableWebTransport:  false,
			},
			wantErr: false,
		},
		{
			name: "valid options - TLS with WebTransport",
			options: &Options{
				EnableTLS:           true,
				EnableTLSClientAuth: false,
				EnableWebTransport:  true,
				WebTransportPort:    "8443",
			},
			wantErr: false,
		},
		{
			name: "valid options - TLS with client auth and WebTransport",
			options: &Options{
				EnableTLS:           true,
				EnableTLSClientAuth: true,
				EnableWebTransport:  true,
				WebTransportPort:    "8443",
			},
			wantErr: false,
		},
		{
			name: "invalid - TLS client auth without TLS",
			options: &Options{
				EnableTLS:           false,
				EnableTLSClientAuth: true,
				EnableWebTransport:  false,
			},
			wantErr: true,
			errMsg:  "TLS client authentication is enabled, but TLS is not enabled",
		},
		{
			name: "invalid - WebTransport without TLS",
			options: &Options{
				EnableTLS:           false,
				EnableTLSClientAuth: false,
				EnableWebTransport:  true,
			},
			wantErr: true,
			errMsg:  "WebTransport requires TLS to be enabled",
		},
		{
			name: "invalid - WebTransport and client auth without TLS",
			options: &Options{
				EnableTLS:           false,
				EnableTLSClientAuth: true,
				EnableWebTransport:  true,
			},
			wantErr: true,
			// Should fail on the first check (TLS client auth)
			errMsg: "TLS client authentication is enabled, but TLS is not enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestOptionsDefaults(t *testing.T) {
	// Test that default values are as expected
	opts := &Options{}

	// Boolean defaults should be false
	if opts.EnableTLS {
		t.Error("EnableTLS should default to false")
	}
	if opts.EnableTLSClientAuth {
		t.Error("EnableTLSClientAuth should default to false")
	}
	if opts.EnableWebTransport {
		t.Error("EnableWebTransport should default to false")
	}
	if opts.PermitWrite {
		t.Error("PermitWrite should default to false")
	}
	if opts.EnableBasicAuth {
		t.Error("EnableBasicAuth should default to false")
	}
	if opts.NoAuth {
		t.Error("NoAuth should default to false")
	}
	if opts.EnableRandomUrl {
		t.Error("EnableRandomUrl should default to false")
	}
	if opts.Once {
		t.Error("Once should default to false")
	}

	// String defaults should be empty
	if opts.Address != "" {
		t.Errorf("Address should default to empty, got %s", opts.Address)
	}
	if opts.Port != "" {
		t.Errorf("Port should default to empty, got %s", opts.Port)
	}
	if opts.WebTransportPort != "" {
		t.Errorf("WebTransportPort should default to empty, got %s", opts.WebTransportPort)
	}

	// Should validate without error
	if err := opts.Validate(); err != nil {
		t.Errorf("Default options should validate, got error: %v", err)
	}
}

func TestOptionsWithTypicalConfiguration(t *testing.T) {
	// Test a typical production configuration
	opts := &Options{
		Address:            "0.0.0.0",
		Port:               "8080",
		Path:               "/",
		PermitWrite:        true,
		EnableBasicAuth:    true,
		Credential:         "admin:password",
		EnableTLS:          true,
		TLSCrtFile:         "/etc/ssl/certs/server.crt",
		TLSKeyFile:         "/etc/ssl/private/server.key",
		EnableWebTransport: true,
		WebTransportPort:   "8443",
		EnableReconnect:    true,
		ReconnectTime:      10,
		MaxConnection:      100,
		EnableWebGL:        true,
	}

	if err := opts.Validate(); err != nil {
		t.Errorf("Typical configuration should validate, got error: %v", err)
	}
}

func TestOptionsWebTransportConfiguration(t *testing.T) {
	// Test various WebTransport configurations
	tests := []struct {
		name    string
		options *Options
		wantErr bool
	}{
		{
			name: "WebTransport with default port",
			options: &Options{
				EnableTLS:          true,
				EnableWebTransport: true,
				WebTransportPort:   "",
			},
			wantErr: false,
		},
		{
			name: "WebTransport with custom port",
			options: &Options{
				EnableTLS:          true,
				EnableWebTransport: true,
				WebTransportPort:   "9443",
			},
			wantErr: false,
		},
		{
			name: "WebTransport disabled doesn't require TLS",
			options: &Options{
				EnableTLS:          false,
				EnableWebTransport: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark options validation
func BenchmarkOptionsValidate(b *testing.B) {
	opts := &Options{
		EnableTLS:          true,
		EnableWebTransport: true,
		WebTransportPort:   "8443",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opts.Validate()
	}
}
