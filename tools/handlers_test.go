package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/roygabriel/mcp-opentext-octane/octane"
)

func req(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

func resultJSON(t *testing.T, result *mcp.CallToolResult) map[string]interface{} {
	t.Helper()
	if result.IsError {
		t.Fatalf("expected success but got error: %+v", result.Content)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content but got none")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(text.Text), &m); err != nil {
		t.Fatalf("failed to unmarshal result JSON: %v", err)
	}
	return m
}

func resultErrText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if !result.IsError {
		t.Fatalf("expected error result but got success: %+v", result.Content)
	}
	if len(result.Content) == 0 {
		return ""
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return text.Text
}

// --- GetEntityHandler ---

func TestGetEntityHandler(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		mock    *MockOctaneService
		wantErr bool
		errMsg  string
	}{
		{
			name: "success",
			args: map[string]interface{}{"id": float64(42)},
			mock: &MockOctaneService{
				GetEntityFunc: func(ctx context.Context, entityType string, id int, fields []string) (*octane.Entity, error) {
					return &octane.Entity{Type: "story", ID: 42, Fields: map[string]interface{}{"name": "Test"}}, nil
				},
			},
		},
		{
			name: "success with fields",
			args: map[string]interface{}{"id": float64(1), "fields": "id,name,phase"},
			mock: &MockOctaneService{
				GetEntityFunc: func(ctx context.Context, entityType string, id int, fields []string) (*octane.Entity, error) {
					if len(fields) != 3 || fields[0] != "id" {
						t.Errorf("expected 3 fields, got %v", fields)
					}
					return &octane.Entity{Type: "story", ID: 1, Fields: map[string]interface{}{"name": "Test"}}, nil
				},
			},
		},
		{
			name:    "missing id",
			args:    map[string]interface{}{},
			mock:    &MockOctaneService{},
			wantErr: true,
			errMsg:  "id is required",
		},
		{
			name:    "zero id",
			args:    map[string]interface{}{"id": float64(0)},
			mock:    &MockOctaneService{},
			wantErr: true,
			errMsg:  "positive integer",
		},
		{
			name:    "negative id",
			args:    map[string]interface{}{"id": float64(-5)},
			mock:    &MockOctaneService{},
			wantErr: true,
			errMsg:  "positive integer",
		},
		{
			name:    "fractional id",
			args:    map[string]interface{}{"id": float64(1.5)},
			mock:    &MockOctaneService{},
			wantErr: true,
			errMsg:  "positive integer",
		},
		{
			name:    "client error",
			args:    map[string]interface{}{"id": float64(1)},
			mock:    newErrMock("not found"),
			wantErr: true,
			errMsg:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := GetEntityHandler(tt.mock, octane.TypeStory)
			result, err := handler(context.Background(), req(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if tt.wantErr {
				errText := resultErrText(t, result)
				if tt.errMsg != "" && !strings.Contains(errText, tt.errMsg) {
					t.Errorf("error = %q, want containing %q", errText, tt.errMsg)
				}
				return
			}
			data := resultJSON(t, result)
			if data["id"] == nil {
				t.Error("expected id in response")
			}
		})
	}
}

// --- ListEntitiesHandler ---

func TestListEntitiesHandler(t *testing.T) {
	successMock := &MockOctaneService{
		ListEntitiesFunc: func(ctx context.Context, entityType string, params octane.QueryParams) (*octane.EntityList, error) {
			return &octane.EntityList{
				Data: []octane.Entity{
					{Type: "story", ID: 1, Fields: map[string]interface{}{"name": "A"}},
					{Type: "story", ID: 2, Fields: map[string]interface{}{"name": "B"}},
				},
				TotalCount:        2,
				ExceedsTotalCount: false,
			}, nil
		},
	}

	tests := []struct {
		name    string
		args    map[string]interface{}
		mock    *MockOctaneService
		wantErr bool
		errMsg  string
	}{
		{
			name: "success with defaults",
			args: map[string]interface{}{},
			mock: successMock,
		},
		{
			name: "with all params",
			args: map[string]interface{}{
				"query":    `"phase={phase.new}"`,
				"fields":   "id,name,phase",
				"order_by": "-creation_time",
				"limit":    float64(25),
				"offset":   float64(10),
			},
			mock: &MockOctaneService{
				ListEntitiesFunc: func(ctx context.Context, entityType string, params octane.QueryParams) (*octane.EntityList, error) {
					if params.Limit != 25 {
						t.Errorf("limit = %d, want 25", params.Limit)
					}
					if params.Offset != 10 {
						t.Errorf("offset = %d, want 10", params.Offset)
					}
					if params.OrderBy != "-creation_time" {
						t.Errorf("order_by = %q, want -creation_time", params.OrderBy)
					}
					if len(params.Fields) != 3 {
						t.Errorf("fields length = %d, want 3", len(params.Fields))
					}
					return &octane.EntityList{Data: []octane.Entity{}, TotalCount: 0}, nil
				},
			},
		},
		{
			name:    "limit too low",
			args:    map[string]interface{}{"limit": float64(0)},
			mock:    successMock,
			wantErr: true,
			errMsg:  "limit must be between 1 and 200",
		},
		{
			name:    "limit too high",
			args:    map[string]interface{}{"limit": float64(201)},
			mock:    successMock,
			wantErr: true,
			errMsg:  "limit must be between 1 and 200",
		},
		{
			name:    "negative offset",
			args:    map[string]interface{}{"offset": float64(-1)},
			mock:    successMock,
			wantErr: true,
			errMsg:  "offset must be >= 0",
		},
		{
			name:    "query too long",
			args:    map[string]interface{}{"query": strings.Repeat("x", maxQueryLength+1)},
			mock:    successMock,
			wantErr: true,
			errMsg:  "query exceeds maximum length",
		},
		{
			name:    "client error",
			args:    map[string]interface{}{},
			mock:    newErrMock("connection failed"),
			wantErr: true,
			errMsg:  "connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ListEntitiesHandler(tt.mock, octane.TypeStory)
			result, err := handler(context.Background(), req(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if tt.wantErr {
				errText := resultErrText(t, result)
				if tt.errMsg != "" && !strings.Contains(errText, tt.errMsg) {
					t.Errorf("error = %q, want containing %q", errText, tt.errMsg)
				}
				return
			}
			data := resultJSON(t, result)
			if data["total_count"] == nil {
				t.Error("expected total_count in response")
			}
			if data["limit"] == nil {
				t.Error("expected limit in response")
			}
		})
	}
}

// --- UpdateEntityHandler ---

func TestUpdateEntityHandler(t *testing.T) {
	successMock := &MockOctaneService{
		GetEntityFunc: func(ctx context.Context, entityType string, id int, fields []string) (*octane.Entity, error) {
			return &octane.Entity{Type: "story", ID: id, Fields: map[string]interface{}{"version_stamp": float64(5)}}, nil
		},
		UpdateEntityFunc: func(ctx context.Context, entityType string, id int, fields map[string]interface{}) (*octane.Entity, error) {
			return &octane.Entity{
				Type:   "story",
				ID:     id,
				Fields: map[string]interface{}{"name": fields["name"], "version_stamp": float64(6)},
			}, nil
		},
	}

	tests := []struct {
		name    string
		args    map[string]interface{}
		mock    *MockOctaneService
		wantErr bool
		errMsg  string
	}{
		{
			name: "success",
			args: map[string]interface{}{
				"id":     float64(42),
				"fields": map[string]interface{}{"name": "Updated"},
			},
			mock: successMock,
		},
		{
			name:    "missing id",
			args:    map[string]interface{}{"fields": map[string]interface{}{"name": "x"}},
			mock:    successMock,
			wantErr: true,
			errMsg:  "id is required",
		},
		{
			name:    "invalid id",
			args:    map[string]interface{}{"id": float64(-1), "fields": map[string]interface{}{"name": "x"}},
			mock:    successMock,
			wantErr: true,
			errMsg:  "positive integer",
		},
		{
			name:    "missing fields",
			args:    map[string]interface{}{"id": float64(1)},
			mock:    successMock,
			wantErr: true,
			errMsg:  "fields is required",
		},
		{
			name:    "fields not object",
			args:    map[string]interface{}{"id": float64(1), "fields": "not an object"},
			mock:    successMock,
			wantErr: true,
			errMsg:  "fields must be a JSON object",
		},
		{
			name: "empty fields",
			args: map[string]interface{}{
				"id":     float64(1),
				"fields": map[string]interface{}{},
			},
			mock:    successMock,
			wantErr: true,
			errMsg:  "at least one field",
		},
		{
			name: "immutable field rejected",
			args: map[string]interface{}{
				"id":     float64(1),
				"fields": map[string]interface{}{"id": 999},
			},
			mock:    successMock,
			wantErr: true,
			errMsg:  "immutable",
		},
		{
			name: "immutable version_stamp rejected",
			args: map[string]interface{}{
				"id":     float64(1),
				"fields": map[string]interface{}{"version_stamp": 10},
			},
			mock:    successMock,
			wantErr: true,
			errMsg:  "immutable",
		},
		{
			name: "client error",
			args: map[string]interface{}{
				"id":     float64(1),
				"fields": map[string]interface{}{"name": "test"},
			},
			mock:    newErrMock("conflict"),
			wantErr: true,
			errMsg:  "conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := UpdateEntityHandler(tt.mock, octane.TypeStory)
			result, err := handler(context.Background(), req(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if tt.wantErr {
				errText := resultErrText(t, result)
				if tt.errMsg != "" && !strings.Contains(errText, tt.errMsg) {
					t.Errorf("error = %q, want containing %q", errText, tt.errMsg)
				}
				return
			}
			data := resultJSON(t, result)
			if data["id"] == nil {
				t.Error("expected id in response")
			}
		})
	}
}

func TestGetEntityHandler_DifferentEntityTypes(t *testing.T) {
	for _, entityType := range []string{octane.TypeStory, octane.TypeDefect, octane.TypeTask, octane.TypeRelease} {
		t.Run(entityType, func(t *testing.T) {
			var capturedType string
			mock := &MockOctaneService{
				GetEntityFunc: func(ctx context.Context, et string, id int, fields []string) (*octane.Entity, error) {
					capturedType = et
					return &octane.Entity{Type: et, ID: id, Fields: map[string]interface{}{}}, nil
				},
			}

			handler := GetEntityHandler(mock, entityType)
			_, err := handler(context.Background(), req(map[string]interface{}{"id": float64(1)}))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if capturedType != entityType {
				t.Errorf("entity type = %q, want %q", capturedType, entityType)
			}
		})
	}
}
