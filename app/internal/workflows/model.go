package workflows

import (
	"context"
	"model"

	"github.com/google/uuid"
	"github.com/mszlu521/thunder/gorms"
	"gorm.io/gorm"
)

type models struct {
	db *gorm.DB
}

func (m *models) deleteWorkflow(ctx context.Context, userId uuid.UUID, id uuid.UUID) error {
	return m.db.WithContext(ctx).Where("id = ? and user_id = ?", id, userId).Delete(&model.Workflow{}).Error
}

func (m *models) getWorkflow(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*model.Workflow, error) {
	var wf model.Workflow
	err := m.db.WithContext(ctx).Where("id = ? and user_id = ?", id, userId).First(&wf).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &wf, err
}

func (m *models) updateWorkflow(ctx context.Context, wf *model.Workflow) error {
	return m.db.WithContext(ctx).Save(wf).Error
}

func (m *models) list(ctx context.Context, f *Filter) ([]*model.Workflow, int64, error) {
	var total int64
	var workflows []*model.Workflow
	if f.Limit == 0 {
		f.Limit = 10
	}
	db := m.db.WithContext(ctx).Model(&model.Workflow{})
	db = db.Where("user_id = ?", f.UserId)
	if f.Name != "" {
		db = db.Where("name LIKE ?", "%"+f.Name+"%")
	}
	db = db.Count(&total)
	db = db.Offset(f.Offset).Limit(f.Limit)
	db = db.Order("id DESC")
	db = db.Find(&workflows)
	return workflows, total, db.Error
}

type Filter struct {
	UserId uuid.UUID
	Name   string
	Limit  int
	Offset int
}

func (m *models) createWorkflow(ctx context.Context, wf *model.Workflow) error {
	return m.db.Create(wf).Error
}

func newModels(db *gorm.DB) *models {
	return &models{
		db: db,
	}
}
