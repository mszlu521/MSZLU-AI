package workflows

import (
	"context"
	"model"

	"github.com/google/uuid"
)

type repository interface {
	createWorkflow(ctx context.Context, m *model.Workflow) error
	list(ctx context.Context, f *Filter) ([]*model.Workflow, int64, error)
	getWorkflow(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*model.Workflow, error)
	updateWorkflow(ctx context.Context, wf *model.Workflow) error
	deleteWorkflow(ctx context.Context, userId uuid.UUID, id uuid.UUID) error
}
