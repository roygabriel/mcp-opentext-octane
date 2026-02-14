package tools

import (
	"strings"
	"testing"
)

func TestValidateEntityID(t *testing.T) {
	tests := []struct {
		name    string
		id      float64
		want    int
		wantErr bool
		errMsg  string
	}{
		{name: "valid", id: 42, want: 42},
		{name: "valid large", id: 100000, want: 100000},
		{name: "one", id: 1, want: 1},
		{name: "zero rejected", id: 0, wantErr: true, errMsg: "positive integer"},
		{name: "negative rejected", id: -5, wantErr: true, errMsg: "positive integer"},
		{name: "fractional rejected", id: 1.5, wantErr: true, errMsg: "positive integer"},
		{name: "negative fractional rejected", id: -1.5, wantErr: true, errMsg: "positive integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateEntityID(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{name: "empty", query: ""},
		{name: "normal", query: `"phase={phase.new}"`},
		{name: "at limit", query: strings.Repeat("x", maxQueryLength)},
		{name: "exceeds limit", query: strings.Repeat("x", maxQueryLength+1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQuery(tt.query)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateFields(t *testing.T) {
	tests := []struct {
		name    string
		fields  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:   "valid single field",
			fields: map[string]interface{}{"name": "new name"},
		},
		{
			name:   "valid multiple fields",
			fields: map[string]interface{}{"name": "new name", "blocked": true},
		},
		{
			name:    "empty fields",
			fields:  map[string]interface{}{},
			wantErr: true,
			errMsg:  "at least one field",
		},
		{
			name:    "immutable id",
			fields:  map[string]interface{}{"id": 1},
			wantErr: true,
			errMsg:  "immutable",
		},
		{
			name:    "immutable type",
			fields:  map[string]interface{}{"type": "story"},
			wantErr: true,
			errMsg:  "immutable",
		},
		{
			name:    "immutable creation_time",
			fields:  map[string]interface{}{"creation_time": "2024-01-01"},
			wantErr: true,
			errMsg:  "immutable",
		},
		{
			name:    "immutable version_stamp",
			fields:  map[string]interface{}{"version_stamp": 5},
			wantErr: true,
			errMsg:  "immutable",
		},
		{
			name:    "immutable workspace_id",
			fields:  map[string]interface{}{"workspace_id": 1},
			wantErr: true,
			errMsg:  "immutable",
		},
		{
			name:    "immutable logical_name",
			fields:  map[string]interface{}{"logical_name": "foo"},
			wantErr: true,
			errMsg:  "immutable",
		},
		{
			name:    "immutable invested_hours",
			fields:  map[string]interface{}{"invested_hours": 5},
			wantErr: true,
			errMsg:  "immutable",
		},
		{
			name:    "immutable subtype",
			fields:  map[string]interface{}{"subtype": "defect"},
			wantErr: true,
			errMsg:  "immutable",
		},
		{
			name:    "immutable last_modified",
			fields:  map[string]interface{}{"last_modified": "2024-01-01"},
			wantErr: true,
			errMsg:  "immutable",
		},
		{
			name:   "mixed valid and immutable",
			fields: map[string]interface{}{"name": "ok", "id": 1},

			wantErr: true,
			errMsg:  "immutable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFields(tt.fields)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
