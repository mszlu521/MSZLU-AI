package agents

import (
	"context"
	"model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository interface {
	createAgent(ctx context.Context, agent *model.Agent) error
	listAgents(ctx context.Context, userID uuid.UUID, filter AgentFilter) ([]*model.Agent, int64, error)
	getAgent(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*model.Agent, error)
	updateAgent(ctx context.Context, agent *model.Agent) error
	deleteAgentTools(ctx context.Context, agentId uuid.UUID) error
	createAgentTools(ctx context.Context, tools []*model.AgentTool) error
	isAgentKnowledgeBaseExist(ctx context.Context, agentId uuid.UUID, knowledgeBaseID uuid.UUID) (bool, error)
	createAgentKnowledgeBase(ctx context.Context, ab *model.AgentKnowledgeBase) error
	deleteAgentKnowledgeBase(ctx context.Context, agentId uuid.UUID, kbId uuid.UUID) error
	getAgentAgent(ctx context.Context, agentId uuid.UUID, marketId uuid.UUID) (*model.AgentAgent, error)
	createAgentAgent(ctx context.Context, agentAgent *model.AgentAgent) error
	deleteAgentAgent(ctx context.Context, agentId uuid.UUID, agentMarketId uuid.UUID) error
	getAgentWorkflow(ctx context.Context, agentId uuid.UUID, workflowID uuid.UUID) (*model.AgentWorkflow, error)
	createAgentWorkflow(ctx context.Context, workflow *model.AgentWorkflow) error
	deleteAgentWorkflow(ctx context.Context, agentId uuid.UUID, workflowId uuid.UUID) error
	transaction(ctx context.Context, f func(tx *gorm.DB) error) error
	deleteAgent(ctx context.Context, id uuid.UUID) error
	deleteAgentKnowledgeBaseByAgentId(ctx context.Context, agentId uuid.UUID) error
	deleteAgentAgentByAgentId(ctx context.Context, agentId uuid.UUID) error
	deleteAgentWorkflowByAgentId(ctx context.Context, agentId uuid.UUID) error
	createSession(ctx context.Context, session *model.ChatSession) error
	listSessions(ctx context.Context, userID uuid.UUID, agentId uuid.UUID) ([]*model.ChatSession, error)
	getSessionMessages(ctx context.Context, sessionId uuid.UUID) ([]*model.ChatMessage, error)
	deleteSession(ctx context.Context, sessionId uuid.UUID) error
	deleteSessionMessages(ctx context.Context, sessionId uuid.UUID) error
	getSession(ctx context.Context, sessionId *uuid.UUID) (*model.ChatSession, error)
	saveChatMessage(ctx context.Context, chatMessage *model.ChatMessage) error
	getAgentSkill(ctx context.Context, agentId uuid.UUID, skillID uuid.UUID) (*model.AgentSkill, error)
	updateAgentSkill(ctx context.Context, agentSkill *model.AgentSkill) error
	saveAgentSkill(ctx context.Context, skill *model.AgentSkill) error
	deleteAgentSkill(ctx context.Context, agentId uuid.UUID, skillID uuid.UUID) error
	deleteAgentTool(ctx context.Context, agentId uuid.UUID, toolId uuid.UUID) error
	getAgentById(ctx context.Context, id uuid.UUID) (*model.Agent, error)
}
