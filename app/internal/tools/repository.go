package tools

import (
	"context"
	"model"

	"github.com/google/uuid"
)

type repository interface {
	getToolByName(ctx context.Context, name string) (*model.Tool, error)
	createTool(ctx context.Context, m *model.Tool) error
	listTools(ctx context.Context, userID uuid.UUID, filter toolFilter) ([]*model.Tool, int64, error)
	getTool(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*model.Tool, error)
	updateTool(ctx context.Context, info *model.Tool) error
	deleteTool(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
	getToolsByIds(ctx context.Context, ids []uuid.UUID) ([]*model.Tool, error)
}
