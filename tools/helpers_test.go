package tools

import (
	"reflect"
	"testing"
)

func TestParseFieldList(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
		key  string
		want []string
	}{
		{
			name: "key missing from args",
			args: map[string]interface{}{},
			key:  "fields",
			want: nil,
		},
		{
			name: "key present but empty string",
			args: map[string]interface{}{"fields": ""},
			key:  "fields",
			want: nil,
		},
		{
			name: "key is non-string type",
			args: map[string]interface{}{"fields": 42},
			key:  "fields",
			want: nil,
		},
		{
			name: "single field",
			args: map[string]interface{}{"fields": "name"},
			key:  "fields",
			want: []string{"name"},
		},
		{
			name: "multiple fields with whitespace",
			args: map[string]interface{}{"fields": " name , status , phase "},
			key:  "fields",
			want: []string{"name", "status", "phase"},
		},
		{
			name: "different key name",
			args: map[string]interface{}{"columns": "a,b"},
			key:  "columns",
			want: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFieldList(tt.args, tt.key)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseFieldList() = %v, want %v", got, tt.want)
			}
		})
	}
}
