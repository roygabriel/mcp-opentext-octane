package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/roygabriel/mcp-opentext-octane/config"
	"github.com/roygabriel/mcp-opentext-octane/octane"
	"github.com/roygabriel/mcp-opentext-octane/tools"
)

var version = "dev"

func main() {
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelInfo)
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		switch strings.ToUpper(lvl) {
		case "DEBUG":
			logLevel.Set(slog.LevelDebug)
		case "WARN":
			logLevel.Set(slog.LevelWarn)
		case "ERROR":
			logLevel.Set(slog.LevelError)
		}
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	client := octane.NewClient(cfg)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	s := server.NewMCPServer(
		"Octane Server",
		version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
		server.WithToolHandlerMiddleware(timeoutMiddleware(60*time.Second)),
		server.WithToolHandlerMiddleware(loggingMiddleware()),
	)

	entityTypes := []struct {
		singular      string
		apiType       string
		displayName   string
		defaultFields string
		updateHint    string
	}{
		{
			"story", octane.TypeStory, "user story",
			"id, name, phase, priority, story_points, owner, feature, release, milestone, sprint, team, blocked, blocked_reason, rank, item_origin, description, creation_time, last_modified",
			"name, description, phase, priority, story_points, owner, feature, release, milestone, sprint, team, blocked, blocked_reason, rank, item_origin",
		},
		{
			"feature", octane.TypeFeature, "feature",
			"id, name, phase, priority, epic, release, milestone, rank, owner, description, creation_time, last_modified",
			"name, description, phase, priority, epic, release, milestone, rank, owner",
		},
		{
			"epic", octane.TypeEpic, "epic",
			"id, name, phase, release, owner, description, creation_time, last_modified",
			"name, description, phase, release, owner",
		},
		{
			"defect", octane.TypeDefect, "defect",
			"id, name, phase, priority, story_points, owner, feature, release, milestone, sprint, team, blocked, blocked_reason, rank, item_origin, description, creation_time, last_modified",
			"name, description, phase, priority, story_points, owner, feature, release, milestone, sprint, team, blocked, blocked_reason, rank, item_origin",
		},
		{
			"task", octane.TypeTask, "task",
			"id, name, phase, owner, backlog_item, sprint, team, estimated_hours, remaining_hours, invested_hours, blocked, blocked_reason, description, creation_time, last_modified",
			"name, description, phase, owner, backlog_item, sprint, team, estimated_hours, remaining_hours, blocked, blocked_reason",
		},
		{
			"release", octane.TypeRelease, "release",
			"id, name, start_date, end_date, creation_time, last_modified",
			"name, start_date, end_date",
		},
		{
			"work_item", octane.TypeWorkItem, "work item",
			"id, name, subtype, phase, priority, owner, release, sprint, team, creation_time, last_modified",
			"name, description, phase, priority, owner, release, sprint, team",
		},
	}

	for _, et := range entityTypes {
		listName := fmt.Sprintf("list_%ss", et.singular)

		getTool := mcp.NewTool(fmt.Sprintf("get_%s", et.singular),
			mcp.WithDescription(fmt.Sprintf("Get a single %s by ID from Octane.\n\nDefault fields: %s.", et.displayName, et.defaultFields)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description(fmt.Sprintf("The %s ID", et.displayName)), mcp.Min(1)),
			mcp.WithString("fields", mcp.Description("Comma-separated field names to return. Omit to use defaults. Reference fields (phase, owner, sprint, etc.) return as objects with id, type, and name properties.")),
		)

		listTool := mcp.NewTool(listName,
			mcp.WithDescription(fmt.Sprintf("Search and list %ss in Octane with filtering, sorting, and pagination.\n\nDefault fields: %s.", et.displayName, et.defaultFields)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("query", mcp.MaxLength(2000), mcp.Description("Octane query filter. Syntax: \"field='value'\" exact match, \"field='val*'\" wildcard, \"field={phase.new}\" reference, \";\" for AND, \"||\" for OR, \"field={null}\" null check, \"field>='2024-01-01T00:00:00Z'\" date comparison. Example: \"phase={phase.new};priority={list_node.priority.high}\".")),
			mcp.WithString("fields", mcp.Description("Comma-separated field names to return. Omit to use defaults. Reference fields (phase, owner, sprint, etc.) return as objects with id, type, and name properties.")),
			mcp.WithString("order_by", mcp.Description("Field name to sort by. Prefix with - for descending. Examples: \"name\", \"-creation_time\", \"id\".")),
			mcp.WithNumber("limit", mcp.Description("Maximum results to return"), mcp.DefaultNumber(50), mcp.Min(1), mcp.Max(200)),
			mcp.WithNumber("offset", mcp.Description("Number of results to skip for pagination"), mcp.DefaultNumber(0), mcp.Min(0)),
		)

		updateTool := mcp.NewTool(fmt.Sprintf("update_%s", et.singular),
			mcp.WithDescription(fmt.Sprintf("Update fields on an existing %s in Octane. Version control is automatic.\n\nUpdatable: %s.\nRead-only (cannot update): id, type, subtype, creation_time, last_modified, version_stamp, invested_hours.", et.displayName, et.updateHint)),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description(fmt.Sprintf("The %s ID to update", et.displayName)), mcp.Min(1)),
			mcp.WithObject("fields", mcp.Required(), mcp.Description("Object of field names to new values. Scalars: {\"name\": \"New title\", \"blocked\": true}. References: {\"phase\": {\"id\": \"phase.new\", \"type\": \"phase\"}, \"owner\": {\"id\": 5001, \"type\": \"workspace_user\"}}. Set null to clear: {\"sprint\": null}. Priority uses: {\"id\": \"list_node.priority.critical\", \"type\": \"list_node\"}.")),
		)

		s.AddTool(getTool, tools.GetEntityHandler(client, et.apiType))
		s.AddTool(listTool, tools.ListEntitiesHandler(client, et.apiType))
		s.AddTool(updateTool, tools.UpdateEntityHandler(client, et.apiType))
	}

	slog.Info("starting Octane MCP server", "version", version)

	stdioServer := server.NewStdioServer(s)
	if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		slog.Error("server error", "error", err)
		return
	}

	slog.Info("server stopped")
}

func timeoutMiddleware(timeout time.Duration) server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return next(ctx, req)
		}
	}
}

func loggingMiddleware() server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			requestID := uuid.New().String()
			tool := req.Params.Name
			logger := slog.With("request_id", requestID, "tool", tool)

			logger.Debug("tool call started")
			start := time.Now()

			result, err := next(ctx, req)
			duration := time.Since(start)

			switch {
			case err != nil:
				logger.Error("tool call failed", "duration_ms", duration.Milliseconds(), "error", err)
			case result != nil && result.IsError:
				logger.Warn("tool call returned error", "duration_ms", duration.Milliseconds())
			default:
				logger.Info("tool call completed", "duration_ms", duration.Milliseconds())
			}

			return result, err
		}
	}
}
