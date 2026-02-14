package octane

import (
	"encoding/json"
	"fmt"
)

const (
	TypeStory    = "stories"
	TypeFeature  = "features"
	TypeEpic     = "epics"
	TypeDefect   = "defects"
	TypeTask     = "tasks"
	TypeRelease  = "releases"
	TypeWorkItem = "work_items"
)

var ValidEntityTypes = map[string]bool{
	TypeStory: true, TypeFeature: true, TypeEpic: true,
	TypeDefect: true, TypeTask: true, TypeRelease: true,
	TypeWorkItem: true,
}

// DefaultFields defines the default field projection per entity type.
var DefaultFields = map[string]string{
	TypeStory:    "id,name,phase,priority,story_points,owner,feature,release,milestone,sprint,team,blocked,blocked_reason,rank,item_origin,description,creation_time,last_modified",
	TypeFeature:  "id,name,phase,priority,epic,release,milestone,rank,owner,description,creation_time,last_modified",
	TypeEpic:     "id,name,phase,release,owner,description,creation_time,last_modified",
	TypeDefect:   "id,name,phase,priority,story_points,owner,feature,release,milestone,sprint,team,blocked,blocked_reason,rank,item_origin,description,creation_time,last_modified",
	TypeTask:     "id,name,phase,owner,backlog_item,sprint,team,estimated_hours,remaining_hours,invested_hours,blocked,blocked_reason,description,creation_time,last_modified",
	TypeRelease:  "id,name,start_date,end_date,creation_time,last_modified",
	TypeWorkItem: "id,name,subtype,phase,priority,owner,release,sprint,team,creation_time,last_modified",
}

// ImmutableFields are fields that cannot be set in updates.
var ImmutableFields = map[string]bool{
	"id": true, "type": true, "subtype": true,
	"creation_time": true, "last_modified": true,
	"version_stamp": true, "workspace_id": true,
	"logical_name": true, "invested_hours": true,
}

// Entity represents an Octane entity with dynamic fields.
type Entity struct {
	Type   string
	ID     int
	Fields map[string]interface{}
}

// MarshalJSON flattens the Entity into a top-level JSON object with type, id, and all Fields.
func (e Entity) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{}, len(e.Fields)+2)
	for k, v := range e.Fields {
		m[k] = v
	}
	m["type"] = e.Type
	m["id"] = e.ID
	return json.Marshal(m)
}

// UnmarshalJSON extracts type and id from top-level JSON, puts everything else in Fields.
func (e *Entity) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if v, ok := raw["type"]; ok {
		if err := json.Unmarshal(v, &e.Type); err != nil {
			return fmt.Errorf("failed to unmarshal type: %w", err)
		}
		delete(raw, "type")
	}

	if v, ok := raw["id"]; ok {
		// id can be float64 from JSON
		var idFloat float64
		if err := json.Unmarshal(v, &idFloat); err != nil {
			return fmt.Errorf("failed to unmarshal id: %w", err)
		}
		e.ID = int(idFloat)
		delete(raw, "id")
	}

	e.Fields = make(map[string]interface{}, len(raw))
	for k, v := range raw {
		var val interface{}
		if err := json.Unmarshal(v, &val); err != nil {
			return fmt.Errorf("failed to unmarshal field %s: %w", k, err)
		}
		e.Fields[k] = val
	}

	return nil
}

// EntityList represents a paginated list of entities from Octane.
type EntityList struct {
	Data              []Entity `json:"data"`
	TotalCount        int      `json:"total_count"`
	ExceedsTotalCount bool     `json:"exceeds_total_count"`
}

// QueryParams holds query parameters for listing entities.
type QueryParams struct {
	Fields  []string
	Query   string
	OrderBy string
	Limit   int
	Offset  int
}
