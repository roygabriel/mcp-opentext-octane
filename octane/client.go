package octane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"

	"github.com/rgabriel/mcp-octane/config"
)

// HTTPBackend abstracts HTTP calls for testability.
type HTTPBackend interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client communicates with the Octane REST API.
type Client struct {
	mu            sync.Mutex
	backend       HTTPBackend
	config        *config.Config
	baseURL       string
	authenticated bool
}

// NewClient creates a Client with a real HTTP backend using a cookie jar.
func NewClient(cfg *config.Config) *Client {
	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{Jar: jar}
	return &Client{
		backend: httpClient,
		config:  cfg,
		baseURL: cfg.URL + cfg.BaseAPIPath(),
	}
}

// NewClientWithBackend creates a Client with a custom backend (for testing).
func NewClientWithBackend(backend HTTPBackend, cfg *config.Config) *Client {
	return &Client{
		backend: backend,
		config:  cfg,
		baseURL: cfg.URL + cfg.BaseAPIPath(),
	}
}

// Close signs out from the Octane session (best-effort).
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.authenticated {
		return nil
	}

	signOutURL := c.config.URL + "/authentication/sign_out"
	req, err := http.NewRequest(http.MethodPost, signOutURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.backend.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	c.authenticated = false
	return nil
}

// GetEntity retrieves a single entity by type and ID.
func (c *Client) GetEntity(ctx context.Context, entityType string, id int, fields []string) (*Entity, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getEntity(ctx, entityType, id, fields)
}

// ListEntities retrieves a list of entities matching the given parameters.
func (c *Client) ListEntities(ctx context.Context, entityType string, params QueryParams) (*EntityList, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.listEntities(ctx, entityType, params)
}

// UpdateEntity updates fields on an entity, handling version_stamp automatically.
func (c *Client) UpdateEntity(ctx context.Context, entityType string, id int, fields map[string]interface{}) (*Entity, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// GET current entity to obtain version_stamp
	current, err := c.getEntity(ctx, entityType, id, []string{"version_stamp"})
	if err != nil {
		return nil, fmt.Errorf("failed to get current entity for version_stamp: %w", err)
	}

	// Build PUT body with version_stamp + user fields
	body := make(map[string]interface{}, len(fields)+1)
	body["version_stamp"] = current.Fields["version_stamp"]
	for k, v := range fields {
		body[k] = v
	}

	path := fmt.Sprintf("/%s/%d", entityType, id)
	resp, err := c.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var entity Entity
	if err := json.NewDecoder(resp.Body).Decode(&entity); err != nil {
		return nil, fmt.Errorf("failed to decode update response: %w", err)
	}
	return &entity, nil
}

func (c *Client) getEntity(ctx context.Context, entityType string, id int, fields []string) (*Entity, error) {
	path := fmt.Sprintf("/%s/%d", entityType, id)

	fieldStr := c.resolveFields(entityType, fields)
	if fieldStr != "" {
		path += "?fields=" + fieldStr
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var entity Entity
	if err := json.NewDecoder(resp.Body).Decode(&entity); err != nil {
		return nil, fmt.Errorf("failed to decode entity response: %w", err)
	}
	return &entity, nil
}

func (c *Client) listEntities(ctx context.Context, entityType string, params QueryParams) (*EntityList, error) {
	path := "/" + entityType

	var queryParts []string

	fieldStr := c.resolveFields(entityType, params.Fields)
	if fieldStr != "" {
		queryParts = append(queryParts, "fields="+fieldStr)
	}
	if params.Query != "" {
		queryParts = append(queryParts, "query="+params.Query)
	}
	if params.OrderBy != "" {
		queryParts = append(queryParts, "order_by="+params.OrderBy)
	}
	if params.Limit > 0 {
		queryParts = append(queryParts, fmt.Sprintf("limit=%d", params.Limit))
	}
	if params.Offset > 0 {
		queryParts = append(queryParts, fmt.Sprintf("offset=%d", params.Offset))
	}

	if len(queryParts) > 0 {
		path += "?" + strings.Join(queryParts, "&")
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var list EntityList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("failed to decode entity list response: %w", err)
	}
	return &list, nil
}

func (c *Client) resolveFields(entityType string, fields []string) string {
	if len(fields) > 0 {
		return strings.Join(fields, ",")
	}
	if defaults, ok := DefaultFields[entityType]; ok {
		return defaults
	}
	return ""
}

func (c *Client) authenticate(ctx context.Context) error {
	var authBody interface{}
	if c.config.UseAPIKey() {
		authBody = map[string]string{
			"client_id":     c.config.ClientID,
			"client_secret": c.config.ClientSecret,
		}
	} else {
		authBody = map[string]string{
			"user":     c.config.Username,
			"password": c.config.Password,
		}
	}

	jsonBody, err := json.Marshal(authBody)
	if err != nil {
		return fmt.Errorf("failed to marshal auth body: %w", err)
	}

	authURL := c.config.URL + "/authentication/sign_in"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.backend.Do(req)
	if err != nil {
		return fmt.Errorf("authentication request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed (status %d): %s", resp.StatusCode, string(body))
	}

	c.authenticated = true
	return nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	if !c.authenticated {
		if err := c.authenticate(ctx); err != nil {
			return nil, err
		}
	}

	resp, err := c.executeRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	// On 401, re-authenticate and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		_ = resp.Body.Close()
		c.authenticated = false
		if err := c.authenticate(ctx); err != nil {
			return nil, err
		}
		resp, err = c.executeRequest(ctx, method, path, body)
		if err != nil {
			return nil, err
		}
	}

	if resp.StatusCode >= 400 {
		defer func() { _ = resp.Body.Close() }()
		respBody, _ := io.ReadAll(resp.Body)

		var octaneErr struct {
			ErrorCode   string `json:"error_code"`
			Description string `json:"description"`
		}
		if json.Unmarshal(respBody, &octaneErr) == nil && octaneErr.Description != "" {
			return nil, fmt.Errorf("octane API error (status %d, code %s): %s", resp.StatusCode, octaneErr.ErrorCode, octaneErr.Description)
		}
		return nil, fmt.Errorf("octane API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return resp, nil
}

func (c *Client) executeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.backend.Do(req)
}
