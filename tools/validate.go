package tools

import (
	"fmt"
)

const maxQueryLength = 2000

var immutableFields = map[string]bool{
	"id": true, "type": true, "subtype": true,
	"creation_time": true, "last_modified": true,
	"version_stamp": true, "workspace_id": true,
	"logical_name": true, "invested_hours": true,
}

func validateEntityID(id float64) (int, error) {
	intID := int(id)
	if float64(intID) != id || intID <= 0 {
		return 0, fmt.Errorf("entity ID must be a positive integer, got %v", id)
	}
	return intID, nil
}

func validateQuery(q string) error {
	if len(q) > maxQueryLength {
		return fmt.Errorf("query exceeds maximum length of %d characters", maxQueryLength)
	}
	return nil
}

func validateFields(fields map[string]interface{}) error {
	if len(fields) == 0 {
		return fmt.Errorf("fields must contain at least one field to update")
	}
	for k := range fields {
		if immutableFields[k] {
			return fmt.Errorf("field %q is immutable and cannot be updated", k)
		}
	}
	return nil
}
