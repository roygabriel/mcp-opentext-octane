package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"unicode"

	"github.com/joho/godotenv"
)

// Config holds the Octane connection configuration.
type Config struct {
	URL           string
	SharedSpaceID string
	WorkspaceID   string
	ClientID      string
	ClientSecret  string
	Username      string
	Password      string
}

// Load reads configuration from environment variables and an optional .env file.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		URL:           os.Getenv("OCTANE_URL"),
		SharedSpaceID: os.Getenv("OCTANE_SHARED_SPACE_ID"),
		WorkspaceID:   os.Getenv("OCTANE_WORKSPACE_ID"),
		ClientID:      os.Getenv("OCTANE_CLIENT_ID"),
		ClientSecret:  os.Getenv("OCTANE_CLIENT_SECRET"),
		Username:      os.Getenv("OCTANE_USERNAME"),
		Password:      os.Getenv("OCTANE_PASSWORD"),
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("OCTANE_URL environment variable is required")
	}

	u, err := url.ParseRequestURI(cfg.URL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("OCTANE_URL must be a valid URL")
	}
	cfg.URL = strings.TrimRight(cfg.URL, "/")

	if cfg.SharedSpaceID == "" {
		return nil, fmt.Errorf("OCTANE_SHARED_SPACE_ID environment variable is required")
	}
	if !isNumeric(cfg.SharedSpaceID) {
		return nil, fmt.Errorf("OCTANE_SHARED_SPACE_ID must be numeric")
	}

	if cfg.WorkspaceID == "" {
		return nil, fmt.Errorf("OCTANE_WORKSPACE_ID environment variable is required")
	}
	if !isNumeric(cfg.WorkspaceID) {
		return nil, fmt.Errorf("OCTANE_WORKSPACE_ID must be numeric")
	}

	hasAPIKey := cfg.ClientID != "" && cfg.ClientSecret != ""
	hasUserPass := cfg.Username != "" && cfg.Password != ""
	if !hasAPIKey && !hasUserPass {
		return nil, fmt.Errorf("either OCTANE_CLIENT_ID+OCTANE_CLIENT_SECRET or OCTANE_USERNAME+OCTANE_PASSWORD must be provided")
	}

	return cfg, nil
}

// BaseAPIPath returns the workspace-scoped API path prefix.
func (c *Config) BaseAPIPath() string {
	return fmt.Sprintf("/api/shared_spaces/%s/workspaces/%s", c.SharedSpaceID, c.WorkspaceID)
}

// UseAPIKey returns true if API key authentication is configured.
func (c *Config) UseAPIKey() bool {
	return c.ClientID != ""
}

func isNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
