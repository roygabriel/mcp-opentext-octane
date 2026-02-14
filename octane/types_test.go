package octane

import (
	"encoding/json"
	"testing"
)

func TestEntityMarshalJSON(t *testing.T) {
	entity := Entity{
		Type: "story",
		ID:   42,
		Fields: map[string]interface{}{
			"name":          "Test Story",
			"version_stamp": float64(5),
			"blocked":       false,
		},
	}

	data, err := json.Marshal(entity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result["type"] != "story" {
		t.Errorf("type = %v, want story", result["type"])
	}
	if result["id"] != float64(42) {
		t.Errorf("id = %v, want 42", result["id"])
	}
	if result["name"] != "Test Story" {
		t.Errorf("name = %v, want Test Story", result["name"])
	}
	if result["version_stamp"] != float64(5) {
		t.Errorf("version_stamp = %v, want 5", result["version_stamp"])
	}
}

func TestEntityMarshalJSON_NilFields(t *testing.T) {
	entity := Entity{Type: "defect", ID: 1}

	data, err := json.Marshal(entity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["type"] != "defect" {
		t.Errorf("type = %v, want defect", result["type"])
	}
	if result["id"] != float64(1) {
		t.Errorf("id = %v, want 1", result["id"])
	}
}

func TestEntityUnmarshalJSON(t *testing.T) {
	input := `{
		"type": "story",
		"id": 1234,
		"name": "My Story",
		"phase": {"type": "phase", "id": "phase.new"},
		"version_stamp": 15,
		"blocked": true
	}`

	var entity Entity
	if err := json.Unmarshal([]byte(input), &entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entity.Type != "story" {
		t.Errorf("Type = %s, want story", entity.Type)
	}
	if entity.ID != 1234 {
		t.Errorf("ID = %d, want 1234", entity.ID)
	}
	if entity.Fields["name"] != "My Story" {
		t.Errorf("name = %v, want My Story", entity.Fields["name"])
	}
	if entity.Fields["blocked"] != true {
		t.Errorf("blocked = %v, want true", entity.Fields["blocked"])
	}
	// type and id should not be in Fields
	if _, ok := entity.Fields["type"]; ok {
		t.Error("type should not be in Fields")
	}
	if _, ok := entity.Fields["id"]; ok {
		t.Error("id should not be in Fields")
	}
}

func TestEntityUnmarshalJSON_InvalidJSON(t *testing.T) {
	var entity Entity
	err := json.Unmarshal([]byte("not json"), &entity)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEntityUnmarshalJSON_InvalidID(t *testing.T) {
	var entity Entity
	err := json.Unmarshal([]byte(`{"id": "not_a_number"}`), &entity)
	if err == nil {
		t.Fatal("expected error for non-numeric id")
	}
}

func TestEntityUnmarshalJSON_InvalidType(t *testing.T) {
	var entity Entity
	err := json.Unmarshal([]byte(`{"type": 123}`), &entity)
	if err == nil {
		t.Fatal("expected error for non-string type")
	}
}

func TestEntityRoundTrip(t *testing.T) {
	original := Entity{
		Type: "feature",
		ID:   99,
		Fields: map[string]interface{}{
			"name":        "Feature X",
			"description": "<p>Hello</p>",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Entity
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type = %s, want %s", decoded.Type, original.Type)
	}
	if decoded.ID != original.ID {
		t.Errorf("ID = %d, want %d", decoded.ID, original.ID)
	}
	if decoded.Fields["name"] != original.Fields["name"] {
		t.Errorf("name = %v, want %v", decoded.Fields["name"], original.Fields["name"])
	}
}

func TestEntityListUnmarshal(t *testing.T) {
	input := `{
		"data": [
			{"type": "story", "id": 1, "name": "A"},
			{"type": "story", "id": 2, "name": "B"}
		],
		"total_count": 42,
		"exceeds_total_count": true
	}`

	var list EntityList
	if err := json.Unmarshal([]byte(input), &list); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list.Data) != 2 {
		t.Errorf("Data length = %d, want 2", len(list.Data))
	}
	if list.TotalCount != 42 {
		t.Errorf("TotalCount = %d, want 42", list.TotalCount)
	}
	if !list.ExceedsTotalCount {
		t.Error("expected ExceedsTotalCount=true")
	}
}
