package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// GetEntityHandler creates a handler that retrieves a single entity by ID.
func GetEntityHandler(client OctaneReader, entityType string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		fields := parseFieldList(args, "fields")

		entity, err := client.GetEntity(ctx, entityType, id, fields)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get %s: %v", entityType, err)), nil
		}

		jsonData, err := json.MarshalIndent(entity, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
