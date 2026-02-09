package agents

import (
	"model"

	"github.com/google/uuid"
)

type ListAgentResponse struct {
	Agents []*model.Agent `json:"agents"`
	Total  int64          `json:"total"`
}

type chatSessionResponse struct {
	ID        uuid.UUID `json:"id"`
	AgentId   uuid.UUID `json:"agentId"`
	Title     string    `json:"title"`
	UserId    uuid.UUID `json:"userId"`
	CreatedAt int64     `json:"createdAt"`
	UpdatedAt int64     `json:"updatedAt"`
}

func toChatSessionResponse(session *model.ChatSession) *chatSessionResponse {
	return &chatSessionResponse{
		ID:        session.ID,
		AgentId:   session.AgentID,
		Title:     session.Title,
		UserId:    session.UserID,
		CreatedAt: session.CreatedAt.UnixMilli(),
		UpdatedAt: session.UpdatedAt.UnixMilli(),
	}
}

type chatMessageResponse struct {
	Id        uuid.UUID `json:"id"`
	SessionId uuid.UUID `json:"sessionId"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt int64     `json:"createdAt"`
}

func toChatMessageResponse(message *model.ChatMessage) *chatMessageResponse {
	return &chatMessageResponse{
		Id:        message.ID,
		SessionId: message.SessionID,
		Role:      message.Role,
		Content:   message.Content,
		CreatedAt: message.CreatedAt.UnixMilli(),
	}
}
func toChatMessageResponses(messages []*model.ChatMessage) []*chatMessageResponse {
	var res []*chatMessageResponse
	for _, message := range messages {
		res = append(res, toChatMessageResponse(message))
	}
	return res
}
