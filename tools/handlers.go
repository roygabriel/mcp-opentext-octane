package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-octane/octane"
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

		var fields []string
		if fieldsStr, ok := args["fields"].(string); ok && fieldsStr != "" {
			fields = strings.Split(fieldsStr, ",")
			for i := range fields {
				fields[i] = strings.TrimSpace(fields[i])
			}
		}

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

// ListEntitiesHandler creates a handler that lists entities with optional filtering and pagination.
func ListEntitiesHandler(client OctaneReader, entityType string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		params := octane.QueryParams{
			Limit:  50,
			Offset: 0,
		}

		if query, ok := args["query"].(string); ok && query != "" {
			if err := validateQuery(query); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			params.Query = query
		}

		if fieldsStr, ok := args["fields"].(string); ok && fieldsStr != "" {
			fields := strings.Split(fieldsStr, ",")
			for i := range fields {
				fields[i] = strings.TrimSpace(fields[i])
			}
			params.Fields = fields
		}

		if orderBy, ok := args["order_by"].(string); ok && orderBy != "" {
			params.OrderBy = orderBy
		}

		if limit, ok := args["limit"].(float64); ok {
			l := int(limit)
			if l < 1 || l > 200 {
				return mcp.NewToolResultError("limit must be between 1 and 200"), nil
			}
			params.Limit = l
		}

		if offset, ok := args["offset"].(float64); ok {
			o := int(offset)
			if o < 0 {
				return mcp.NewToolResultError("offset must be >= 0"), nil
			}
			params.Offset = o
		}

		result, err := client.ListEntities(ctx, entityType, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list %s: %v", entityType, err)), nil
		}

		response := map[string]interface{}{
			"data":                result.Data,
			"total_count":         result.TotalCount,
			"exceeds_total_count": result.ExceedsTotalCount,
			"limit":               params.Limit,
			"offset":              params.Offset,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

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
