package tools

import "strings"

// parseFieldList extracts a comma-separated list from args[key], trims
// whitespace around each element, and returns the resulting slice.
// Returns nil when the key is absent or the value is empty.
func parseFieldList(args map[string]interface{}, key string) []string {
	raw, ok := args[key].(string)
	if !ok || raw == "" {
		return nil
	}

	fields := strings.Split(raw, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return fields
}
