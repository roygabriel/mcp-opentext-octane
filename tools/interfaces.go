package tools

import (
	"context"

	"github.com/roygabriel/mcp-opentext-octane/octane"
)

// OctaneReader defines read-only Octane operations.
type OctaneReader interface {
	GetEntity(ctx context.Context, entityType string, id int, fields []string) (*octane.Entity, error)
	ListEntities(ctx context.Context, entityType string, params octane.QueryParams) (*octane.EntityList, error)
}

// OctaneWriter defines mutating Octane operations.
type OctaneWriter interface {
	UpdateEntity(ctx context.Context, entityType string, id int, fields map[string]interface{}) (*octane.Entity, error)
}

// OctaneService combines all Octane operations.
type OctaneService interface {
	OctaneReader
	OctaneWriter
}
