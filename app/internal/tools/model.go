package tools

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

func (m *models) getToolsByIds(ctx context.Context, ids []uuid.UUID) ([]*model.Tool, error) {
	var tools []*model.Tool
	return tools, m.db.WithContext(ctx).Where("id in ?", ids).Find(&tools).Error
}

func (m *models) deleteTool(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	return m.db.WithContext(ctx).Where("id = ? and creator_id=?", id, userID).Delete(&model.Tool{}).Error
}

func (m *models) getTool(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*model.Tool, error) {
	var tool model.Tool
	err := m.db.WithContext(ctx).Where("id = ? and creator_id=?", id, userID).First(&tool).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &tool, err
}

func (m *models) updateTool(ctx context.Context, info *model.Tool) error {
	return m.db.WithContext(ctx).Updates(info).Error
}

func (m *models) listTools(ctx context.Context, userID uuid.UUID, filter toolFilter) ([]*model.Tool, int64, error) {
	var tools []*model.Tool
	var count int64
	query := m.db.WithContext(ctx).Model(&model.Tool{}).Where("creator_id = ?", userID)
	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}
	if filter.ToolType != "" {
		query = query.Where("tool_type = ?", filter.ToolType)
	}
	query = query.Count(&count)
	if filter.Limit != 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}
	err := query.Find(&tools).Error
	return tools, count, err
}

type toolFilter struct {
	Name     string
	ToolType model.ToolType
	Limit    int
	Offset   int
}

func (m *models) getToolByName(ctx context.Context, name string) (*model.Tool, error) {
	var tool model.Tool
	err := m.db.WithContext(ctx).Where("name = ?", name).First(&tool).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &tool, err
}

func (m *models) createTool(ctx context.Context, tool *model.Tool) error {
	return m.db.WithContext(ctx).Create(tool).Error
}

func newModels(db *gorm.DB) *models {
	return &models{
		db: db,
	}
}
