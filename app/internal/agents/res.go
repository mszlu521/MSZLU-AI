package agents

import "model"

type ListAgentResponse struct {
	Agents []*model.Agent `json:"agents"`
	Total  int64          `json:"total"`
}
