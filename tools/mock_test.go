package tools

import (
	"context"
	"fmt"

	"github.com/rgabriel/mcp-octane/octane"
)

// MockOctaneService implements OctaneService for testing.
type MockOctaneService struct {
	GetEntityFunc    func(ctx context.Context, entityType string, id int, fields []string) (*octane.Entity, error)
	ListEntitiesFunc func(ctx context.Context, entityType string, params octane.QueryParams) (*octane.EntityList, error)
	UpdateEntityFunc func(ctx context.Context, entityType string, id int, fields map[string]interface{}) (*octane.Entity, error)

	CallCount int
}

func (m *MockOctaneService) GetEntity(ctx context.Context, entityType string, id int, fields []string) (*octane.Entity, error) {
	m.CallCount++
	return m.GetEntityFunc(ctx, entityType, id, fields)
}

func (m *MockOctaneService) ListEntities(ctx context.Context, entityType string, params octane.QueryParams) (*octane.EntityList, error) {
	m.CallCount++
	return m.ListEntitiesFunc(ctx, entityType, params)
}

func (m *MockOctaneService) UpdateEntity(ctx context.Context, entityType string, id int, fields map[string]interface{}) (*octane.Entity, error) {
	m.CallCount++
	return m.UpdateEntityFunc(ctx, entityType, id, fields)
}

func newErrMock(msg string) *MockOctaneService {
	err := fmt.Errorf("%s", msg)
	return &MockOctaneService{
		GetEntityFunc: func(ctx context.Context, entityType string, id int, fields []string) (*octane.Entity, error) {
			return nil, err
		},
		ListEntitiesFunc: func(ctx context.Context, entityType string, params octane.QueryParams) (*octane.EntityList, error) {
			return nil, err
		},
		UpdateEntityFunc: func(ctx context.Context, entityType string, id int, fields map[string]interface{}) (*octane.Entity, error) {
			return nil, err
		},
	}
}
