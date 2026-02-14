package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// UpdateEntityHandler creates a handler that updates fields on an existing entity.
func UpdateEntityHandler(client OctaneService, entityType string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		idFloat, ok := args["id"].(float64)
		if !ok {
			return mcp.NewToolResultError("id is required"), nil
		}
		id, err := validateEntityID(idFloat)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		fieldsRaw, ok := args["fields"]
		if !ok {
			return mcp.NewToolResultError("fields is required"), nil
		}
		fields, ok := fieldsRaw.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("fields must be a JSON object"), nil
		}

		if err := validateFields(fields); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		entity, err := client.UpdateEntity(ctx, entityType, id, fields)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to update %s: %v", entityType, err)), nil
		}

		jsonData, err := json.MarshalIndent(entity, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
