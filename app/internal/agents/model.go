package agents

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

func (m *models) updateAgent(ctx context.Context, agent *model.Agent) error {
	return m.db.WithContext(ctx).Updates(agent).Error
}

func (m *models) getAgent(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*model.Agent, error) {
	var agent model.Agent
	err := m.db.WithContext(ctx).Where("id = ? and creator_id = ? ", id, userID).First(&agent).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &agent, err
}

func (m *models) listAgents(ctx context.Context, userID uuid.UUID, filter AgentFilter) ([]*model.Agent, int64, error) {
	var agents []*model.Agent
	var count int64
	query := m.db.WithContext(ctx).Model(&model.Agent{})
	query = query.Where("creator_id = ?", userID)
	if filter.Name != "" {
		query = query.Where("name like ?", "%"+filter.Name+"%")
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	query = query.Count(&count)
	query = query.Limit(filter.Limit).Offset(filter.Offset)
	return agents, count, query.Find(&agents).Error
}

type AgentFilter struct {
	Name   string
	Status model.AgentStatus
	Limit  int
	Offset int
}

func (m *models) createAgent(ctx context.Context, agent *model.Agent) error {
	return m.db.WithContext(ctx).Create(agent).Error
}

func newModels(db *gorm.DB) *models {
	return &models{
		db: db,
	}
}
