package config

import (
	"testing"
	"time"
)

func TestServerConfigBind(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(c *Config)
		expectError bool
		expected    ServerConfig
	}{
		{
			name:        "default values only",
			setup:       func(_ *Config) {},
			expectError: false,
			expected: ServerConfig{
				Addr:         ":8080",
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
		},
		{
			name: "custom values with valid TLS",
			setup: func(c *Config) {
				c.Set("addr", ":9090")
				c.Set("read_timeout", "60s")
				c.Set("write_timeout", "45s")
				c.Set("tls.enabled", true)
				c.Set("tls.cert", "/path/to/cert.pem")
				c.Set("tls.key", "/path/to/key.pem")
			},
			expectError: false,
			expected: ServerConfig{
				Addr:         ":9090",
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 45 * time.Second,
				TLS: ServerTLS{
					Enabled: true,
					Cert:    "/path/to/cert.pem",
					Key:     "/path/to/key.pem",
				},
			},
		},
		{
			name: "TLS enabled without cert files should fail",
			setup: func(c *Config) {
				c.Set("tls.enabled", true)
				// Missing tls.cert and tls.key - should fail validation
			},
			expectError: true,
		},
		{
			name: "invalid timeout format",
			setup: func(c *Config) {
				c.Set("read_timeout", "invalid-duration")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			tt.setup(c)

			var config ServerConfig
			err := c.Bind("", &config)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				t.Logf("Got expected error: %v", err)
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the configuration values
			if config.Addr != tt.expected.Addr {
				t.Errorf("expected Addr=%q, got %q", tt.expected.Addr, config.Addr)
			}

			if config.ReadTimeout != tt.expected.ReadTimeout {
				t.Errorf("expected ReadTimeout=%v, got %v", tt.expected.ReadTimeout, config.ReadTimeout)
			}

			if config.WriteTimeout != tt.expected.WriteTimeout {
				t.Errorf("expected WriteTimeout=%v, got %v", tt.expected.WriteTimeout, config.WriteTimeout)
			}

			if config.TLS.Enabled != tt.expected.TLS.Enabled {
				t.Errorf("expected TLS.Enabled=%v, got %v", tt.expected.TLS.Enabled, config.TLS.Enabled)
			}

			if config.TLS.Cert != tt.expected.TLS.Cert {
				t.Errorf("expected TLS.Cert=%q, got %q", tt.expected.TLS.Cert, config.TLS.Cert)
			}

			if config.TLS.Key != tt.expected.TLS.Key {
				t.Errorf("expected TLS.Key=%q, got %q", tt.expected.TLS.Key, config.TLS.Key)
			}
		})
	}
}

func TestServerConfigWithPrefix(t *testing.T) {
	c := New()
	c.Set("server.addr", ":9090")
	c.Set("server.read_timeout", "60s")
	c.Set("server.tls.enabled", true)
	c.Set("server.tls.cert", "/path/to/cert.pem")

	var config ServerConfig
	err := c.Bind("server", &config)
	if err != nil {
		t.Fatalf("failed to bind with prefix: %v", err)
	}

	if config.Addr != ":9090" {
		t.Errorf("expected Addr=:9090, got %q", config.Addr)
	}

	if config.ReadTimeout != 60*time.Second {
		t.Errorf("expected ReadTimeout=60s, got %v", config.ReadTimeout)
	}

	if !config.TLS.Enabled {
		t.Error("expected TLS.Enabled=true, got false")
	}

	if config.TLS.Cert != "/path/to/cert.pem" {
		t.Errorf("expected TLS.Cert=/path/to/cert.pem, got %q", config.TLS.Cert)
	}
}

func TestServerConfigDurationValidation(t *testing.T) {
	c := New()

	// Set extremely long timeouts that might be unreasonable
	c.Set("read_timeout", "24h")    // 24 hours
	c.Set("write_timeout", "8760h") // 1 year

	var config ServerConfig
	err := c.Bind("", &config)
	if err != nil {
		t.Fatalf("unexpected error for long durations: %v", err)
	}

	// The bind should succeed even with long durations
	// (unless you have specific validation rules for duration limits)
	if config.ReadTimeout != 24*time.Hour {
		t.Errorf("expected ReadTimeout=24h, got %v", config.ReadTimeout)
	}
}

// ServerConfig struct for testing
type ServerConfig struct {
	Addr         string        `config:"addr" check:"default=:8080"`
	ReadTimeout  time.Duration `config:"read_timeout" check:"default=30s"`
	WriteTimeout time.Duration `config:"write_timeout" check:"default=30s"`
	TLS          ServerTLS     `config:"tls"`
}

type ServerTLS struct {
	Enabled bool   `config:"enabled" check:"default=false"`
	Cert    string `config:"cert" check:"required"`
	Key     string `config:"key"`
}
