package agents

import (
	"model"

	"github.com/google/uuid"
)

type CreateAgentReq struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Status      model.AgentStatus `json:"status"`
}

type SearchAgentReq struct {
	Params struct {
		Name     string            `json:"name"`
		Status   model.AgentStatus `json:"status"`
		Page     int               `json:"page"`
		PageSize int               `json:"pageSize"`
	} `json:"params"`
}

type UpdateAgentReq struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Status          model.AgentStatus `json:"status"`
	SystemPrompt    string            `json:"systemPrompt"`
	ModelProvider   string            `json:"modelProvider"`
	ModelName       string            `json:"modelName"`
	ModelParameters model.JSON        `json:"modelParameters"`
	OpeningDialogue string            `json:"openingDialogue"`
}
type AgentMessageReq struct {
	AgentID   uuid.UUID `json:"agentId"`
	Message   string    `json:"message"`
	SessionId uuid.UUID `json:"sessionId,omitempty"`
}

type UpdateAgentToolReq struct {
	Tools []ToolItem `json:"tools"`
}

type ToolItem struct {
	ID   uuid.UUID `json:"id"`
	Type string    `json:"type"`
}

type addAgentKnowledgeBaseReq struct {
	KnowledgeBaseID uuid.UUID `json:"kb_id"`
}
