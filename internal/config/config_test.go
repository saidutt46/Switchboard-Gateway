package config

import (
	"os"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid development config",
			config: Config{
				Environment: "development",
				ServerHost:  "localhost",
				ServerPort:  8080,
				LogLevel:    "info",
				LogFormat:   "console",
				Database: DatabaseConfig{
					DSN:          "postgres://localhost:5432/test",
					MaxOpenConns: 25,
					MaxIdleConns: 5,
				},
			},
			wantErr: false,
		},
		{
			name: "valid production config",
			config: Config{
				Environment: "production",
				ServerHost:  "0.0.0.0",
				ServerPort:  8080,
				LogLevel:    "error",
				LogFormat:   "json",
				Database: DatabaseConfig{
					DSN:          "postgres://localhost:5432/prod",
					MaxOpenConns: 100,
					MaxIdleConns: 10,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid environment",
			config: Config{
				Environment: "invalid",
				ServerPort:  8080,
				LogLevel:    "info",
				LogFormat:   "json",
				Database: DatabaseConfig{
					DSN:          "postgres://localhost:5432/test",
					MaxOpenConns: 25,
					MaxIdleConns: 5,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port - too low",
			config: Config{
				Environment: "development",
				ServerPort:  0,
				LogLevel:    "info",
				LogFormat:   "json",
				Database: DatabaseConfig{
					DSN:          "postgres://localhost:5432/test",
					MaxOpenConns: 25,
					MaxIdleConns: 5,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			config: Config{
				Environment: "development",
				ServerPort:  70000,
				LogLevel:    "info",
				LogFormat:   "json",
				Database: DatabaseConfig{
					DSN:          "postgres://localhost:5432/test",
					MaxOpenConns: 25,
					MaxIdleConns: 5,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: Config{
				Environment: "development",
				ServerPort:  8080,
				LogLevel:    "trace",
				LogFormat:   "json",
				Database: DatabaseConfig{
					DSN:          "postgres://localhost:5432/test",
					MaxOpenConns: 25,
					MaxIdleConns: 5,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			config: Config{
				Environment: "development",
				ServerPort:  8080,
				LogLevel:    "info",
				LogFormat:   "xml",
				Database: DatabaseConfig{
					DSN:          "postgres://localhost:5432/test",
					MaxOpenConns: 25,
					MaxIdleConns: 5,
				},
			},
			wantErr: true,
		},
		{
			name: "max idle conns greater than max open conns",
			config: Config{
				Environment: "development",
				ServerPort:  8080,
				LogLevel:    "info",
				LogFormat:   "json",
				Database: DatabaseConfig{
					DSN:          "postgres://localhost:5432/test",
					MaxOpenConns: 10,
					MaxIdleConns: 20,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	cfg := Config{Environment: "development"}
	if !cfg.IsDevelopment() {
		t.Error("expected IsDevelopment to return true")
	}

	cfg.Environment = "production"
	if cfg.IsDevelopment() {
		t.Error("expected IsDevelopment to return false")
	}
}

func TestConfig_IsProduction(t *testing.T) {
	cfg := Config{Environment: "production"}
	if !cfg.IsProduction() {
		t.Error("expected IsProduction to return true")
	}

	cfg.Environment = "development"
	if cfg.IsProduction() {
		t.Error("expected IsProduction to return false")
	}
}

func TestConfig_ServerAddress(t *testing.T) {
	cfg := Config{
		ServerHost: "localhost",
		ServerPort: 8080,
	}

	expected := "localhost:8080"
	if cfg.ServerAddress() != expected {
		t.Errorf("expected %s, got %s", expected, cfg.ServerAddress())
	}
}

func TestConfig_Load(t *testing.T) {
	// Set required environment variable
	os.Setenv("POSTGRES_DSN", "postgres://localhost:5432/test")
	defer os.Unsetenv("POSTGRES_DSN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load to succeed, got error: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config to be non-nil")
	}

	// Check defaults
	if cfg.Environment != "development" {
		t.Errorf("expected default environment to be 'development', got %s", cfg.Environment)
	}

	if cfg.ServerPort != 8080 {
		t.Errorf("expected default port to be 8080, got %d", cfg.ServerPort)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level to be 'info', got %s", cfg.LogLevel)
	}
}
