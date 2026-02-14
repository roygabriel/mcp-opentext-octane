package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name: "valid API key auth",
			env: map[string]string{
				"OCTANE_URL":             "https://octane.example.com",
				"OCTANE_SHARED_SPACE_ID": "1001",
				"OCTANE_WORKSPACE_ID":    "2002",
				"OCTANE_CLIENT_ID":       "my_client",
				"OCTANE_CLIENT_SECRET":   "my_secret",
			},
		},
		{
			name: "valid user/password auth",
			env: map[string]string{
				"OCTANE_URL":             "https://octane.example.com",
				"OCTANE_SHARED_SPACE_ID": "1001",
				"OCTANE_WORKSPACE_ID":    "2002",
				"OCTANE_USERNAME":        "user@example.com",
				"OCTANE_PASSWORD":        "password123",
			},
		},
		{
			name: "both auth modes prefers API key",
			env: map[string]string{
				"OCTANE_URL":             "https://octane.example.com",
				"OCTANE_SHARED_SPACE_ID": "1001",
				"OCTANE_WORKSPACE_ID":    "2002",
				"OCTANE_CLIENT_ID":       "my_client",
				"OCTANE_CLIENT_SECRET":   "my_secret",
				"OCTANE_USERNAME":        "user@example.com",
				"OCTANE_PASSWORD":        "password123",
			},
		},
		{
			name: "trailing slash stripped",
			env: map[string]string{
				"OCTANE_URL":             "https://octane.example.com/",
				"OCTANE_SHARED_SPACE_ID": "1001",
				"OCTANE_WORKSPACE_ID":    "2002",
				"OCTANE_CLIENT_ID":       "my_client",
				"OCTANE_CLIENT_SECRET":   "my_secret",
			},
		},
		{
			name:    "missing URL",
			env:     map[string]string{},
			wantErr: "OCTANE_URL environment variable is required",
		},
		{
			name: "invalid URL",
			env: map[string]string{
				"OCTANE_URL": "not-a-url",
			},
			wantErr: "OCTANE_URL must be a valid URL",
		},
		{
			name: "missing shared space ID",
			env: map[string]string{
				"OCTANE_URL": "https://octane.example.com",
			},
			wantErr: "OCTANE_SHARED_SPACE_ID environment variable is required",
		},
		{
			name: "non-numeric shared space ID",
			env: map[string]string{
				"OCTANE_URL":             "https://octane.example.com",
				"OCTANE_SHARED_SPACE_ID": "abc",
			},
			wantErr: "OCTANE_SHARED_SPACE_ID must be numeric",
		},
		{
			name: "missing workspace ID",
			env: map[string]string{
				"OCTANE_URL":             "https://octane.example.com",
				"OCTANE_SHARED_SPACE_ID": "1001",
			},
			wantErr: "OCTANE_WORKSPACE_ID environment variable is required",
		},
		{
			name: "non-numeric workspace ID",
			env: map[string]string{
				"OCTANE_URL":             "https://octane.example.com",
				"OCTANE_SHARED_SPACE_ID": "1001",
				"OCTANE_WORKSPACE_ID":    "abc",
			},
			wantErr: "OCTANE_WORKSPACE_ID must be numeric",
		},
		{
			name: "missing both auth modes",
			env: map[string]string{
				"OCTANE_URL":             "https://octane.example.com",
				"OCTANE_SHARED_SPACE_ID": "1001",
				"OCTANE_WORKSPACE_ID":    "2002",
			},
			wantErr: "either OCTANE_CLIENT_ID+OCTANE_CLIENT_SECRET or OCTANE_USERNAME+OCTANE_PASSWORD must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars
			for _, key := range []string{
				"OCTANE_URL", "OCTANE_SHARED_SPACE_ID", "OCTANE_WORKSPACE_ID",
				"OCTANE_CLIENT_ID", "OCTANE_CLIENT_SECRET",
				"OCTANE_USERNAME", "OCTANE_PASSWORD",
			} {
				t.Setenv(key, "")
			}
			// Set test-specific values
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := Load()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				if cfg != nil {
					t.Fatal("expected nil config on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoad_TrailingSlashStripped(t *testing.T) {
	t.Setenv("OCTANE_URL", "https://octane.example.com///")
	t.Setenv("OCTANE_SHARED_SPACE_ID", "1001")
	t.Setenv("OCTANE_WORKSPACE_ID", "2002")
	t.Setenv("OCTANE_CLIENT_ID", "id")
	t.Setenv("OCTANE_CLIENT_SECRET", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.URL != "https://octane.example.com" {
		t.Errorf("URL = %q, want trailing slashes stripped", cfg.URL)
	}
}

func TestLoad_BothAuthModesAPIKeyWins(t *testing.T) {
	t.Setenv("OCTANE_URL", "https://octane.example.com")
	t.Setenv("OCTANE_SHARED_SPACE_ID", "1001")
	t.Setenv("OCTANE_WORKSPACE_ID", "2002")
	t.Setenv("OCTANE_CLIENT_ID", "my_client")
	t.Setenv("OCTANE_CLIENT_SECRET", "my_secret")
	t.Setenv("OCTANE_USERNAME", "user@example.com")
	t.Setenv("OCTANE_PASSWORD", "password123")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.UseAPIKey() {
		t.Error("expected UseAPIKey() to return true when both modes provided")
	}
}

func TestBaseAPIPath(t *testing.T) {
	cfg := &Config{
		SharedSpaceID: "1001",
		WorkspaceID:   "2002",
	}
	want := "/api/shared_spaces/1001/workspaces/2002"
	if got := cfg.BaseAPIPath(); got != want {
		t.Errorf("BaseAPIPath() = %q, want %q", got, want)
	}
}

func TestUseAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
		want     bool
	}{
		{"with client ID", "my_client", true},
		{"without client ID", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{ClientID: tt.clientID}
			if got := cfg.UseAPIKey(); got != tt.want {
				t.Errorf("UseAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
