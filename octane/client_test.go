package octane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/roygabriel/mcp-opentext-octane/config"
)

// MockHTTPBackend implements HTTPBackend for testing.
type MockHTTPBackend struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPBackend) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func testConfig() *config.Config {
	return &config.Config{
		URL:           "https://octane.example.com",
		SharedSpaceID: "1001",
		WorkspaceID:   "2002",
		ClientID:      "test_client",
		ClientSecret:  "test_secret",
	}
}

func testConfigUserPass() *config.Config {
	return &config.Config{
		URL:           "https://octane.example.com",
		SharedSpaceID: "1001",
		WorkspaceID:   "2002",
		Username:      "user@example.com",
		Password:      "password123",
	}
}

func jsonResponse(status int, body interface{}) *http.Response {
	data, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(string(data))),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func authThenRespond(cfg *config.Config, resp *http.Response) func(req *http.Request) (*http.Response, error) {
	authDone := false
	return func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/authentication/sign_in") {
			authDone = true
			return jsonResponse(http.StatusOK, map[string]string{"status": "ok"}), nil
		}
		if !authDone {
			return jsonResponse(http.StatusUnauthorized, nil), nil
		}
		return resp, nil
	}
}

func TestAuthenticate_APIKey(t *testing.T) {
	cfg := testConfig()
	var capturedBody map[string]string

	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/authentication/sign_in") {
				body, _ := io.ReadAll(req.Body)
				json.Unmarshal(body, &capturedBody)
				return jsonResponse(http.StatusOK, nil), nil
			}
			return jsonResponse(http.StatusOK, nil), nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	err := client.authenticate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !client.authenticated {
		t.Error("expected authenticated to be true")
	}
	if capturedBody["client_id"] != "test_client" {
		t.Errorf("expected client_id=test_client, got %s", capturedBody["client_id"])
	}
	if capturedBody["client_secret"] != "test_secret" {
		t.Errorf("expected client_secret=test_secret, got %s", capturedBody["client_secret"])
	}
}

func TestAuthenticate_UserPassword(t *testing.T) {
	cfg := testConfigUserPass()
	var capturedBody map[string]string

	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			json.Unmarshal(body, &capturedBody)
			return jsonResponse(http.StatusOK, nil), nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	err := client.authenticate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedBody["user"] != "user@example.com" {
		t.Errorf("expected user=user@example.com, got %s", capturedBody["user"])
	}
	if capturedBody["password"] != "password123" {
		t.Errorf("expected password=password123, got %s", capturedBody["password"])
	}
}

func TestAuthenticate_Failure(t *testing.T) {
	cfg := testConfig()
	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusUnauthorized, map[string]string{
				"description": "bad credentials",
			}), nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	err := client.authenticate(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected auth failure message, got: %v", err)
	}
}

func TestAuthenticate_NetworkError(t *testing.T) {
	cfg := testConfig()
	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		},
	}

	client := NewClientWithBackend(mock, cfg)
	err := client.authenticate(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("expected network error, got: %v", err)
	}
}

func TestGetEntity_Success(t *testing.T) {
	cfg := testConfig()
	entity := map[string]interface{}{
		"type": "story", "id": float64(1234),
		"name": "Test Story", "version_stamp": float64(5),
	}

	mock := &MockHTTPBackend{
		DoFunc: authThenRespond(cfg, jsonResponse(http.StatusOK, entity)),
	}

	client := NewClientWithBackend(mock, cfg)
	result, err := client.GetEntity(context.Background(), TypeStory, 1234, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "story" {
		t.Errorf("Type = %s, want story", result.Type)
	}
	if result.ID != 1234 {
		t.Errorf("ID = %d, want 1234", result.ID)
	}
	if result.Fields["name"] != "Test Story" {
		t.Errorf("name = %v, want Test Story", result.Fields["name"])
	}
}

func TestGetEntity_WithFields(t *testing.T) {
	cfg := testConfig()
	var capturedURL string

	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/authentication/") {
				return jsonResponse(http.StatusOK, nil), nil
			}
			capturedURL = req.URL.String()
			return jsonResponse(http.StatusOK, map[string]interface{}{
				"type": "story", "id": float64(1),
			}), nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	_, err := client.GetEntity(context.Background(), TypeStory, 1, []string{"id", "name", "phase"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedURL, "fields=id,name,phase") {
		t.Errorf("expected fields param in URL, got: %s", capturedURL)
	}
}

func TestGetEntity_DefaultFields(t *testing.T) {
	cfg := testConfig()
	var capturedURL string

	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/authentication/") {
				return jsonResponse(http.StatusOK, nil), nil
			}
			capturedURL = req.URL.String()
			return jsonResponse(http.StatusOK, map[string]interface{}{
				"type": "story", "id": float64(1),
			}), nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	_, err := client.GetEntity(context.Background(), TypeStory, 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedURL, "fields="+DefaultFields[TypeStory]) {
		t.Errorf("expected default fields in URL, got: %s", capturedURL)
	}
}

func TestListEntities_Success(t *testing.T) {
	cfg := testConfig()
	list := map[string]interface{}{
		"data": []map[string]interface{}{
			{"type": "story", "id": float64(1), "name": "Story A"},
			{"type": "story", "id": float64(2), "name": "Story B"},
		},
		"total_count":         float64(2),
		"exceeds_total_count": false,
	}

	mock := &MockHTTPBackend{
		DoFunc: authThenRespond(cfg, jsonResponse(http.StatusOK, list)),
	}

	client := NewClientWithBackend(mock, cfg)
	result, err := client.ListEntities(context.Background(), TypeStory, QueryParams{
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", result.TotalCount)
	}
	if len(result.Data) != 2 {
		t.Errorf("Data length = %d, want 2", len(result.Data))
	}
}

func TestListEntities_QueryParams(t *testing.T) {
	cfg := testConfig()
	var capturedURL string

	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/authentication/") {
				return jsonResponse(http.StatusOK, nil), nil
			}
			capturedURL = req.URL.String()
			return jsonResponse(http.StatusOK, map[string]interface{}{
				"data":                []interface{}{},
				"total_count":         float64(0),
				"exceeds_total_count": false,
			}), nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	_, err := client.ListEntities(context.Background(), TypeStory, QueryParams{
		Fields:  []string{"id", "name"},
		Query:   `"phase={phase.new}"`,
		OrderBy: "-creation_time",
		Limit:   25,
		Offset:  10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []string{"fields=id,name", "query=", "order_by=-creation_time", "limit=25", "offset=10"}
	for _, c := range checks {
		if !strings.Contains(capturedURL, c) {
			t.Errorf("expected URL to contain %q, got: %s", c, capturedURL)
		}
	}
}

func TestUpdateEntity_Success(t *testing.T) {
	cfg := testConfig()
	callCount := 0

	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/authentication/") {
				return jsonResponse(http.StatusOK, nil), nil
			}
			callCount++
			if req.Method == http.MethodGet {
				// Return current entity with version_stamp
				return jsonResponse(http.StatusOK, map[string]interface{}{
					"type": "story", "id": float64(1234),
					"name": "Old Name", "version_stamp": float64(15),
				}), nil
			}
			if req.Method == http.MethodPut {
				// Verify version_stamp is in the body
				body, _ := io.ReadAll(req.Body)
				var putBody map[string]interface{}
				json.Unmarshal(body, &putBody)
				if putBody["version_stamp"] != float64(15) {
					t.Errorf("expected version_stamp=15, got %v", putBody["version_stamp"])
				}
				return jsonResponse(http.StatusOK, map[string]interface{}{
					"type": "story", "id": float64(1234),
					"name": "New Name", "version_stamp": float64(16),
				}), nil
			}
			return jsonResponse(http.StatusMethodNotAllowed, nil), nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	result, err := client.UpdateEntity(context.Background(), TypeStory, 1234, map[string]interface{}{
		"name": "New Name",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Fields["name"] != "New Name" {
		t.Errorf("name = %v, want New Name", result.Fields["name"])
	}
}

func TestDoRequest_401Retry(t *testing.T) {
	cfg := testConfig()
	authCount := 0
	requestCount := 0

	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/authentication/sign_in") {
				authCount++
				return jsonResponse(http.StatusOK, nil), nil
			}
			requestCount++
			if requestCount == 1 {
				return jsonResponse(http.StatusUnauthorized, nil), nil
			}
			return jsonResponse(http.StatusOK, map[string]interface{}{
				"type": "story", "id": float64(1),
			}), nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	_, err := client.GetEntity(context.Background(), TypeStory, 1, []string{"id"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authCount != 2 {
		t.Errorf("authCount = %d, want 2 (initial + retry)", authCount)
	}
}

func TestDoRequest_HTTPErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       interface{}
		wantMsg    string
	}{
		{
			name:       "400 bad request",
			statusCode: http.StatusBadRequest,
			body:       map[string]string{"error_code": "bad_request", "description": "invalid query syntax"},
			wantMsg:    "invalid query syntax",
		},
		{
			name:       "403 forbidden",
			statusCode: http.StatusForbidden,
			body:       map[string]string{"error_code": "forbidden", "description": "access denied"},
			wantMsg:    "access denied",
		},
		{
			name:       "404 not found",
			statusCode: http.StatusNotFound,
			body:       map[string]string{"error_code": "not_found", "description": "entity not found"},
			wantMsg:    "entity not found",
		},
		{
			name:       "409 conflict",
			statusCode: http.StatusConflict,
			body:       map[string]string{"error_code": "conflict", "description": "version stamp mismatch"},
			wantMsg:    "version stamp mismatch",
		},
		{
			name:       "500 server error",
			statusCode: http.StatusInternalServerError,
			body:       "internal server error",
			wantMsg:    "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			mock := &MockHTTPBackend{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					if strings.Contains(req.URL.Path, "/authentication/") {
						return jsonResponse(http.StatusOK, nil), nil
					}
					return jsonResponse(tt.statusCode, tt.body), nil
				},
			}

			client := NewClientWithBackend(mock, cfg)
			_, err := client.GetEntity(context.Background(), TypeStory, 1, []string{"id"})
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestUpdateEntity_GetFails(t *testing.T) {
	cfg := testConfig()
	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/authentication/") {
				return jsonResponse(http.StatusOK, nil), nil
			}
			return jsonResponse(http.StatusNotFound, map[string]string{
				"error_code":  "not_found",
				"description": "entity not found",
			}), nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	_, err := client.UpdateEntity(context.Background(), TypeStory, 9999, map[string]interface{}{
		"name": "test",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "entity not found") {
		t.Errorf("error = %q, want containing entity not found", err.Error())
	}
}

func TestClose_Authenticated(t *testing.T) {
	cfg := testConfig()
	var signOutCalled bool

	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/authentication/sign_out") {
				signOutCalled = true
				return jsonResponse(http.StatusOK, nil), nil
			}
			return jsonResponse(http.StatusOK, nil), nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	client.authenticated = true
	err := client.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !signOutCalled {
		t.Error("expected sign_out to be called")
	}
}

func TestClose_NotAuthenticated(t *testing.T) {
	cfg := testConfig()
	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			t.Fatal("should not make any HTTP calls")
			return nil, nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	err := client.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestURLConstruction(t *testing.T) {
	cfg := testConfig()
	var capturedURL string

	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/authentication/") {
				return jsonResponse(http.StatusOK, nil), nil
			}
			capturedURL = req.URL.String()
			return jsonResponse(http.StatusOK, map[string]interface{}{
				"type": "story", "id": float64(1),
			}), nil
		},
	}

	client := NewClientWithBackend(mock, cfg)
	_, err := client.GetEntity(context.Background(), TypeStory, 42, []string{"id"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "https://octane.example.com/api/shared_spaces/1001/workspaces/2002/stories/42?fields=id"
	if capturedURL != expected {
		t.Errorf("URL = %s, want %s", capturedURL, expected)
	}
}

func TestContextCancellation(t *testing.T) {
	cfg := testConfig()
	mock := &MockHTTPBackend{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("request canceled: %w", req.Context().Err())
		},
	}

	client := NewClientWithBackend(mock, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetEntity(ctx, TypeStory, 1, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewClient(t *testing.T) {
	cfg := testConfig()
	client := NewClient(cfg)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	expectedBase := "https://octane.example.com/api/shared_spaces/1001/workspaces/2002"
	if client.baseURL != expectedBase {
		t.Errorf("baseURL = %s, want %s", client.baseURL, expectedBase)
	}
}
