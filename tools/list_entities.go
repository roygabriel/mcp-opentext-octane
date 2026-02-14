package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/roygabriel/mcp-opentext-octane/octane"
)

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

		params.Fields = parseFieldList(args, "fields")

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
