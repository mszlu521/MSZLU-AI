package agents

import (
	"context"
	"model"

	"github.com/google/uuid"
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
}
