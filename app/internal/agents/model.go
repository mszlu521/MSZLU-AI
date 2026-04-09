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

func (m *models) getAgentById(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	var agent model.Agent
	err := m.db.WithContext(ctx).Where("id = ?", id).First(&agent).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &agent, err
}

func (m *models) deleteAgentTool(ctx context.Context, agentId uuid.UUID, toolId uuid.UUID) error {
	return m.db.WithContext(ctx).Where("agent_id = ? and tool_id = ?", agentId, toolId).Delete(&model.AgentTool{}).Error
}

func (m *models) deleteAgentSkill(ctx context.Context, agentId uuid.UUID, skillID uuid.UUID) error {
	return m.db.WithContext(ctx).Where("agent_id = ? and skill_id = ?", agentId, skillID).Delete(&model.AgentSkill{}).Error
}

func (m *models) getAgentSkill(ctx context.Context, agentId uuid.UUID, skillID uuid.UUID) (*model.AgentSkill, error) {
	var agentSkill model.AgentSkill
	err := m.db.WithContext(ctx).Where("agent_id = ? and skill_id = ?", agentId, skillID).First(&agentSkill).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &agentSkill, err
}

func (m *models) updateAgentSkill(ctx context.Context, agentSkill *model.AgentSkill) error {
	return m.db.WithContext(ctx).Updates(agentSkill).Error
}

func (m *models) saveAgentSkill(ctx context.Context, skill *model.AgentSkill) error {
	return m.db.WithContext(ctx).Create(skill).Error
}

func (m *models) getSession(ctx context.Context, sessionId *uuid.UUID) (*model.ChatSession, error) {
	var session model.ChatSession
	err := m.db.WithContext(ctx).Where("id = ?", sessionId).First(&session).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &session, err
}

func (m *models) saveChatMessage(ctx context.Context, chatMessage *model.ChatMessage) error {
	return m.db.WithContext(ctx).Create(chatMessage).Error
}

func (m *models) deleteSession(ctx context.Context, sessionId uuid.UUID) error {
	return m.db.WithContext(ctx).Where("id = ?", sessionId).Unscoped().Delete(&model.ChatSession{}).Error
}

func (m *models) deleteSessionMessages(ctx context.Context, sessionId uuid.UUID) error {
	return m.db.WithContext(ctx).Where("session_id = ?", sessionId).Unscoped().Delete(&model.ChatMessage{}).Error
}

func (m *models) getSessionMessages(ctx context.Context, sessionId uuid.UUID) ([]*model.ChatMessage, error) {
	var messages []*model.ChatMessage
	err := m.db.WithContext(ctx).Where("session_id = ?", sessionId).Find(&messages).Error
	return messages, err
}

func (m *models) listSessions(ctx context.Context, userID uuid.UUID, agentId uuid.UUID) ([]*model.ChatSession, error) {
	var sessions []*model.ChatSession
	err := m.db.WithContext(ctx).Where("agent_id = ?", agentId).Find(&sessions).Error
	return sessions, err
}

func (m *models) createSession(ctx context.Context, session *model.ChatSession) error {
	return m.db.WithContext(ctx).Create(session).Error
}

func (m *models) deleteAgent(ctx context.Context, id uuid.UUID) error {
	return m.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Agent{}).Error
}

func (m *models) deleteAgentKnowledgeBaseByAgentId(ctx context.Context, agentId uuid.UUID) error {
	return m.db.WithContext(ctx).Where("agent_id = ?", agentId).Delete(&model.AgentKnowledgeBase{}).Error
}

func (m *models) deleteAgentAgentByAgentId(ctx context.Context, agentId uuid.UUID) error {
	return m.db.WithContext(ctx).Where("agent_id = ?", agentId).Delete(&model.AgentAgent{}).Error
}

func (m *models) deleteAgentWorkflowByAgentId(ctx context.Context, agentId uuid.UUID) error {
	return m.db.WithContext(ctx).Where("agent_id = ?", agentId).Delete(&model.AgentWorkflow{}).Error
}

func (m *models) transaction(ctx context.Context, f func(tx *gorm.DB) error) error {
	return m.db.WithContext(ctx).Transaction(f)
}

func (m *models) deleteAgentWorkflow(ctx context.Context, agentId uuid.UUID, workflowId uuid.UUID) error {
	return m.db.WithContext(ctx).Where("agent_id = ? and workflow_id = ?", agentId, workflowId).Delete(&model.AgentWorkflow{}).Error
}

func (m *models) getAgentWorkflow(ctx context.Context, agentId uuid.UUID, workflowID uuid.UUID) (*model.AgentWorkflow, error) {
	var wf model.AgentWorkflow
	err := m.db.WithContext(ctx).Where("agent_id = ? and workflow_id = ?", agentId, workflowID).First(&wf).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &wf, err
}

func (m *models) createAgentWorkflow(ctx context.Context, workflow *model.AgentWorkflow) error {
	return m.db.WithContext(ctx).Create(workflow).Error
}

func (m *models) deleteAgentAgent(ctx context.Context, agentId uuid.UUID, agentMarketId uuid.UUID) error {
	return m.db.WithContext(ctx).Where("agent_id = ? and agent_market_id = ?", agentId, agentMarketId).Delete(&model.AgentAgent{}).Error
}

func (m *models) getAgentAgent(ctx context.Context, agentId uuid.UUID, marketId uuid.UUID) (*model.AgentAgent, error) {
	var modelAgent model.AgentAgent
	err := m.db.WithContext(ctx).Where("agent_id = ? and agent_market_id = ?", agentId, marketId).First(&modelAgent).Error
	if gorms.IsRecordNotFoundError(err) {
		return nil, nil
	}
	return &modelAgent, nil
}

func (m *models) createAgentAgent(ctx context.Context, agentAgent *model.AgentAgent) error {
	return m.db.WithContext(ctx).Create(agentAgent).Error
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
		Preload("Agents").
		Preload("Workflows").
		Preload("Skills").
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
