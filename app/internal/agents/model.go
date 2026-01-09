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

func (m *models) deleteAgentKnowledgeBase(ctx context.Context, agentId uuid.UUID, kbId uuid.UUID) error {
	return m.db.WithContext(ctx).Where("agent_id = ? and knowledge_base_id = ?", agentId, kbId).Delete(&model.AgentKnowledgeBase{}).Error
}

func (m *models) isAgentKnowledgeBaseExist(ctx context.Context, agentId uuid.UUID, knowledgeBaseID uuid.UUID) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).Model(&model.AgentKnowledgeBase{}).Where("agent_id = ? and knowledge_base_id = ?", agentId, knowledgeBaseID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *models) createAgentKnowledgeBase(ctx context.Context, ab *model.AgentKnowledgeBase) error {
	return m.db.WithContext(ctx).Create(ab).Error
}

func (m *models) deleteAgentTools(ctx context.Context, agentId uuid.UUID) error {
	return m.db.WithContext(ctx).Where("agent_id = ?", agentId).Delete(&model.AgentTool{}).Error
}

func (m *models) createAgentTools(ctx context.Context, tools []*model.AgentTool) error {
	return m.db.WithContext(ctx).CreateInBatches(tools, len(tools)).Error
}

func (m *models) updateAgent(ctx context.Context, agent *model.Agent) error {
	return m.db.WithContext(ctx).Updates(agent).Error
}

func (m *models) getAgent(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*model.Agent, error) {
	var agent model.Agent
	err := m.db.WithContext(ctx).
		Preload("Tools").
		Preload("KnowledgeBases").
		Where("id = ? and creator_id = ? ", id, userID).First(&agent).Error
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
